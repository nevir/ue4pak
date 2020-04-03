package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Vilsol/ue4pak/parser"
	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"

	"github.com/gobwas/glob"
	"github.com/spf13/cobra"
)

var assets *[]string

func init() {
	assets = extractCmd.Flags().StringSliceP("assets", "a", []string{}, "Comma-separated list of asset paths to extract. (supports glob) (required)")
	format = extractCmd.Flags().StringP("format", "f", "json", "Output format type")
	output = extractCmd.Flags().StringP("output", "o", "extracted.json", "Output file (or directory if --split)")
	split = extractCmd.Flags().Bool("split", false, "Whether output should be split into a file per asset")
	compact = extractCmd.Flags().Bool("compact", false, "Whether output should omit verbose metadata and statistics")
	pretty = extractCmd.Flags().Bool("pretty", false, "Whether to output in a pretty format")

	extractCmd.MarkFlagRequired("assets")

	rootCmd.AddCommand(extractCmd)
}

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract provided asset paths",
	Run: func(cmd *cobra.Command, args []string) {
		color.NoColor = false

		paks, err := filepath.Glob(cmd.Flag("pak").Value.String())

		if err != nil {
			panic(err)
		}

		patterns := make([]glob.Glob, len(*assets))
		for i, asset := range *assets {
			patterns[i] = glob.MustCompile(asset)
		}

		results := make([]*parser.PakEntrySet, 0)

		for _, f := range paks {
			fmt.Println("Parsing file:", f)

			file, err := os.OpenFile(f, os.O_RDONLY, 0644)

			if err != nil {
				panic(err)
			}

			shouldProcess := func(name string) bool {
				for _, pattern := range patterns {
					if pattern.Match(name) {
						return true
					}
				}

				return false
			}

			p := parser.NewParser(file)
			p.ProcessPak(shouldProcess, func(name string, entry *parser.PakEntrySet, _ *parser.PakFile) {
				if *split {
					destination := filepath.Join(*output, entry.ExportRecord.FileName+"."+*format)
					err := os.MkdirAll(filepath.Dir(destination), 0755)
					if err != nil {
						panic(err)
					}

					log.Infof("Writing Result: %s\n", destination)
					resultBytes := formatEntry(entry)
					err = ioutil.WriteFile(destination, resultBytes, 0644)
					if err != nil {
						panic(err)
					}
				} else {
					results = append(results, entry)
				}
			})
		}

		if !*split {
			resultBytes := formatEntries(results)
			err = ioutil.WriteFile(*output, resultBytes, 0644)
		}

		if err != nil {
			panic(err)
		}
	},
}

func formatEntry(entry *parser.PakEntrySet) []byte {
	if *compact {
		return marshalResults(parser.MakeCompactEntry(entry))
	} else {
		return marshalResults(entry)
	}
}

func formatEntries(entries []*parser.PakEntrySet) []byte {
	if *compact {
		compactEntries := make([]*parser.CompactEntry, len(entries))
		for i, entry := range entries {
			compactEntries[i] = parser.MakeCompactEntry(entry)
		}
		return marshalResults(compactEntries)
	} else {
		return marshalResults(entries)
	}
}

func marshalResults(result interface{}) []byte {
	var resultBytes []byte
	var err error

	if *format == "json" {
		if *pretty {
			resultBytes, err = json.MarshalIndent(result, "", "  ")
		} else {
			resultBytes, err = json.Marshal(result)
		}

		if err != nil {
			panic(err)
		}
	} else {
		panic("Unknown output format: " + *format)
	}

	return resultBytes
}
