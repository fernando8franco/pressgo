package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	iloveapi "github.com/fernando8franco/i-love-api-golang"
	"github.com/fernando8franco/pressgo/internal/config"
	"github.com/olekukonko/tablewriter"
)

func HandlerCredentials(s *state, cmd command) error {
	fs := flag.NewFlagSet(cmd.Name, flag.ExitOnError)

	var (
		help     = fs.Bool(initHelpFlag, false, "Show help message")
		add      = fs.Bool("add", false, "Add new credential -add <id> <key>")
		delete   = fs.Bool("delete", false, "Delete credential -delete <id>")
		activate = fs.Bool("activate", false, "Activate credential -activate <id>")
	)
	fs.Parse(cmd.Arguments)

	if *help {
		fs.Usage()
		return nil
	}

	if *add {
		cmd.Arguments = fs.Args()
		if len(cmd.Arguments) != 2 {
			fmt.Printf("Error: -add requires exactly two arguments: id and key.\nUsage: pressgo -add <id> <key>\n")
			os.Exit(1)
		}

		id, err := addCredential(s, cmd)
		if err != nil {
			return err
		}
		fmt.Printf("The credential with id: %v was added\n", id)
		return nil
	}

	if *delete {
		cmd.Arguments = fs.Args()
		if len(cmd.Arguments) != 1 {
			fmt.Printf("Error: -delete requires exactly one argument: id.\nUsage: pressgo -delete <id>\n")
			os.Exit(1)
		}

		id, err := deleteCredential(s, cmd)
		if err != nil {
			return err
		}
		fmt.Printf("The credential with id: %v was deleted\n", id)
		return nil
	}

	if *activate {
		cmd.Arguments = fs.Args()
		if len(cmd.Arguments) != 1 {
			fmt.Printf("Error: -activate requires exactly one argument: id.\nUsage: pressgo -activate <id>\n")
			os.Exit(1)
		}

		id := cmd.Arguments[0]
		s.cfg.ActivateCredential(id)
		fmt.Printf("The credential with id: %v was activated\n", id)
		return nil
	}

	credentials := s.cfg.GetCredentials()
	if len(credentials) == 0 {
		fs.Usage()
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"ID", "Key", "Credits", "Status"})
	table.Bulk(credentials)
	table.Render()

	return nil
}

func addCredential(s *state, cmd command) (string, error) {
	id := cmd.Arguments[0]
	key := cmd.Arguments[1]

	token, credits, err := validateCredential(s, key)
	if err != nil {
		return "", fmt.Errorf("Error in validating the credential")
	}

	err = s.cfg.AddCredential(id, config.CreateCredential(key, token, credits))
	if err != nil {
		return "", err
	}

	return id, nil
}

func validateCredential(s *state, key string) (string, int, error) {
	api := iloveapi.NewClient(s.client)
	err := api.GenerateToken(context.Background(), key)
	if err != nil {
		return "", 0, err
	}

	start, err := api.Start(context.Background(), iloveapi.StartParams{Tool: toolCompress, Region: region})
	if err != nil {
		return "", 0, err
	}

	return api.GetToken(), start.RemainingCredits, nil
}

func deleteCredential(s *state, cmd command) (string, error) {
	id := cmd.Arguments[0]
	err := s.cfg.DeleteCredential(id)
	if err != nil {
		return "", err
	}

	return id, nil
}
