/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/phux/apijc/app"

	"github.com/spf13/cobra"
)

var (
	urlFile    string
	baseDomain string
	newDomain  string
	rateLimit  float64
	outputFile string
	headerFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "apijc",
	Short: "compare json responses across two domains",
	Long:  `compare json responses across two domains.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("\nStarting with rate limit: %f/second\n\n", rateLimit)
		urls, err := app.LoadURLsFromFile(urlFile)
		if err != nil {
			log.Fatalf("Error: %s\n", err)
		}

		headers, err := loadHeadersFromFile()
		if err != nil {
			log.Fatalln(err)
		}

		parser := app.NewURLParser()
		a := app.NewApp(
			baseDomain,
			newDomain,
			parser,
			rateLimit,
			headers,
		)
		a.AddURLs(*urls)

		err = a.Run()
		if err != nil {
			log.Fatalf("Error: %s\n", err)
		}

		if len(a.Results.Findings) == 0 {
			log.Println("All targets matched!")

			return
		}

		findings, err := json.MarshalIndent(a.Results.Findings, "", "  ")
		if err != nil {
			log.Fatalf(err.Error())
		}

		if outputFile == "" {
			log.Println("Findings:")
			for _, finding := range a.Results.Findings {
				log.Printf("%s\nError: %s\nDiff: "+finding.Diff, finding.URL, finding.Error)
			}
		} else {
			err = os.WriteFile(outputFile, findings, 0o644)
			if err != nil {
				log.Fatalln(err)
			}

			log.Fatalf("Written findings to %s", outputFile)
		}

		log.Fatalf("Finished - %d findings", len(a.Results.Findings))
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&urlFile, "urlFile", "", "[required] JSON file with relative paths, HTTP method, ...")
	rootCmd.MarkFlagRequired("urlFile")
	rootCmd.Flags().StringVar(&baseDomain, "baseDomain", "", "[required] baseDomain: domain for the left side of the comparison")
	rootCmd.MarkFlagRequired("baseDomain")
	rootCmd.Flags().StringVar(&newDomain, "newDomain", "", "[required] newDomain: domain for the right side of the comparison")
	rootCmd.MarkFlagRequired("newDomain")
	rootCmd.Flags().Float64Var(&rateLimit, "rateLimit", 1, "[optional] rate limit of requests / second")
	rootCmd.Flags().StringVar(&outputFile, "outputFile", "", "[optional] outputFile: path to write the findings to if > 0 findings (default: \"\" -> writing to stdout)")
	rootCmd.Flags().StringVar(&headerFile, "headerFile", "", "[optional] headerFile: provide (additional) header key-value pairs via a JSON object (string: string). Applied to every request")
}

func loadHeadersFromFile() (map[string]string, error) {
	if headerFile == "" {
		return map[string]string{}, nil
	}

	content, err := ioutil.ReadFile(headerFile)
	if err != nil {
		return nil, err
	}

	var headers map[string]string
	err = json.Unmarshal(content, &headers)
	if err != nil {
		return nil, err
	}

	return headers, nil
}
