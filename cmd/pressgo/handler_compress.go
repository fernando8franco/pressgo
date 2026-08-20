package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fernando8franco/pressgo/pkg/pdfs"
	"github.com/fernando8franco/pressgo/pkg/slug"
	"golang.org/x/sync/errgroup"

	iloveapi "github.com/fernando8franco/i-love-api-golang"
)

var (
	ErrInsufficientCredits = errors.New("insufficient credits for this operation")
)

type CreditsUpdate struct {
	Credits   int
	Requested time.Time
	Err       error
}

func HandlerCompress(s *state, cmd command) error {
	fs := flag.NewFlagSet(cmd.Name, flag.ExitOnError)
	var (
		help = fs.Bool(initHelpFlag, false, "Show help message")
		init = fs.Bool(initFlag, false, "Create config file -init <title> <author>\nIf title == 'base', all filenames default to the base name.")
		// noInit = fs.Bool(noInitFlag, false, "Compress files without config file -no-init")
	)
	fs.Parse(cmd.Arguments)

	if *help {
		fs.Usage()
		return nil
	}

	cfgFile := path.Join(s.wdir, configFileName)

	if *init {
		cmd.Arguments = fs.Args()
		if len(cmd.Arguments) != 2 {
			fmt.Printf("Error: -init requires exactly two arguments: title and author.\nUsage: pressgo -init <title> <author>\n")
			os.Exit(1)
		}

		err := initConfig(s, cmd, cfgFile)
		if err != nil {
			return err
		}
		return nil
	}

	if _, err := os.Stat(cfgFile); errors.Is(err, os.ErrNotExist) {
		fmt.Printf("PDF's config file not found\nTry -help first.\n")
		return nil
	}

	pdfs, err := getConfigPdfsFile(cfgFile)
	if err != nil {
		return err
	}

	client := &http.Client{}
	credential, err := s.cfg.GetCredential()
	if err != nil {
		return err
	}
	ila := iloveapi.NewClient(client, credential.Key)

	compressPDFDirPath := filepath.Join(s.wdir, compressDir)
	if _, err := os.Stat(compressPDFDirPath); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(compressPDFDirPath, 0755)
		if err != nil {
			return err
		}
	}

	pdfsChannel := make(chan PDFsConfig)
	var wg errgroup.Group
	for range 3 {
		wg.Go(func() error {
			for pdf := range pdfsChannel {
				err := func(pdf PDFsConfig) error {
					fmt.Println("Compressing:", pdf.Filename)

					startResponse, err := callWithRetry(s, ila, func() (iloveapi.StartResponse, error) {
						return ila.Start(context.Background(), iloveapi.StartParams{Tool: toolCompress, Region: region})
					})
					if err != nil {
						return err
					}

					pdfFile := filepath.Join(pdf.Path, pdf.Filename)
					file, err := os.Open(pdfFile)
					if err != nil {
						return err
					}
					defer file.Close()

					uploadResponse, err := callWithRetry(s, ila, func() (iloveapi.UploadResponse, error) {
						return ila.Upload(context.Background(), iloveapi.UploadParams{
							Server:   startResponse.Server,
							Task:     startResponse.Task,
							File:     file,
							FileName: pdf.Filename,
						})
					})
					if err != nil {
						return err
					}

					_, err = callWithRetry(s, ila, func() (iloveapi.ProcessResponse, error) {
						return ila.Process(context.Background(), iloveapi.ProcessParams{
							Server: startResponse.Server,
							Task:   startResponse.Task,
							Tool:   toolCompress,
							Files: []iloveapi.Files{
								{
									ServerFileName: uploadResponse.ServerFilename,
									FileName:       pdf.Filename,
								},
							},
							Meta: iloveapi.Meta{
								Title:  pdf.Title,
								Author: pdf.Author,
							},
						})
					})
					if err != nil {
						return err
					}

					rel, err := filepath.Rel(s.wdir, pdf.Path)
					if err != nil {
						return err
					}
					compressPDFDirPath := filepath.Join(s.wdir, compressDir, rel)
					if _, err := os.Stat(compressPDFDirPath); errors.Is(err, os.ErrNotExist) {
						err := os.Mkdir(compressPDFDirPath, 0755)
						if err != nil {
							return err
						}
					}
					compressPDFFilePath := filepath.Join(compressPDFDirPath, pdf.NewName)
					out, err := os.Create(compressPDFFilePath)
					if err != nil {
						return err
					}
					defer out.Close()

					dowloadResponse, err := callWithRetry(s, ila, func() (io.ReadCloser, error) {
						return ila.Download(context.Background(), iloveapi.DowloadParams{
							Server: startResponse.Server,
							Task:   startResponse.Task,
						})
					})
					if err != nil {
						return err
					}
					defer dowloadResponse.Close()

					_, err = io.Copy(out, dowloadResponse)
					if err != nil {
						return err
					}

					fmt.Println(pdf.Filename, "--- Compressed correctly")
					return nil
				}(pdf)

				if err != nil {
					return err
				}
			}
			return nil
		})
	}

	go func() {
		defer close(pdfsChannel)
		for _, info := range pdfs {
			pdfsChannel <- info
		}
	}()

	if err := wg.Wait(); err != nil {
		return err
	}

	fmt.Println("All pdfs were compressed correctly")

	err = os.Remove(cfgFile)
	if err != nil {
		return err
	}

	return nil
}

func callWithRetry[T any](s *state, ila *iloveapi.Client, apiFunc func() (T, error)) (T, error) {
	response, err := apiFunc()
	if err == nil {
		return response, nil
	}

	var apiErr *iloveapi.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode() == 404 {
		fmt.Printf("%+v\n", response)
	}
	if errors.As(err, &apiErr) && apiErr.StatusCode() == 401 {
		err = updateConfigToken(s, ila)
		if err != nil {
			var zero T
			return zero, fmt.Errorf("failed to update the config token: %w", err)
		}

		return apiFunc()
	}

	return response, err
}

func updateConfigToken(s *state, ila *iloveapi.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	token := ila.GetToken()
	if token != "" && s.cfg.GetTokenCredential() == token {
		return nil
	}

	err := ila.GenerateToken(context.Background())
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	token = ila.GetToken()

	if cred, err := s.cfg.GetCredential(); err != nil {
		return err
	} else if cred.Token == token {
		return nil
	}

	return s.cfg.UpdateTokenCredential(token)
}

func initConfig(s *state, cmd command, cfgFile string) error {
	if _, err := os.Stat(cfgFile); !errors.Is(err, os.ErrNotExist) {
		fmt.Println("The config pdfs file is already created")
		fmt.Print("You want to delete it and create another one? (y/n) ")
		var answer string
		fmt.Scan(&answer)

		lowAnswer := strings.ToLower(answer)
		if lowAnswer != "y" && lowAnswer != "yes" {
			return nil
		}
	}

	title := cmd.Arguments[0]
	author := cmd.Arguments[1]

	err := generateConfigPdfsFile(s.wdir, cfgFile, title, author)
	if err != nil {
		return fmt.Errorf("error generating config pdfs file: %v", err)
	}

	return nil
}

type PDFsConfig struct {
	Filename string `json:"filename"`
	NewName  string `json:"new_name"`
	Title    string `json:"title"`
	Author   string `json:"author"`
	Path     string `json:"path"`
}

func generateConfigPdfsFile(pdfDir, configPDFsFilePath, title, author string) error {
	pdfsInfo, err := getPDFs(pdfDir, title, author)
	if err != nil {
		return err
	}

	cfgPDFsFile, err := os.Create(configPDFsFilePath)
	if err != nil {
		return err
	}
	defer cfgPDFsFile.Close()

	encoder := json.NewEncoder(cfgPDFsFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(pdfsInfo); err != nil {
		return err
	}

	return nil
}

func getPDFs(pdfDir, title, author string) ([]PDFsConfig, error) {
	pdfs, err := pdfs.GetFromDir(pdfDir)
	if err != nil {
		return nil, err
	}

	if len(pdfs) == 0 {
		return nil, fmt.Errorf("No pdfs found.")
	}

	pdfsInfo := []PDFsConfig{}
	for _, pdf := range pdfs {
		path := filepath.Dir(pdf)
		base := filepath.Base(pdf)
		ext := filepath.Ext(pdf)
		filenameWithoutExt := strings.TrimSuffix(base, ext)
		newFilename := slug.Create(filenameWithoutExt) + pdfExt

		var metaTitle string
		if title == titleFilename {
			metaTitle = filenameWithoutExt
		} else {
			metaTitle = title
		}

		pdfsInfo = append(pdfsInfo, PDFsConfig{
			Filename: base,
			NewName:  newFilename,
			Title:    metaTitle,
			Author:   author,
			Path:     path,
		})
	}

	return pdfsInfo, nil
}

func getConfigPdfsFile(cfgPDFsFile string) ([]PDFsConfig, error) {
	configPdfsFile, err := os.Open(cfgPDFsFile)
	if err != nil {
		return nil, err
	}
	defer configPdfsFile.Close()

	var cfg []PDFsConfig
	if err := json.NewDecoder(configPdfsFile).Decode(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
