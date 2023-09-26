package cmd

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(lsCmd)
}

func readDir(dirname string, maxDepth int, indent string) error {
	maxDepth = maxDepth - 1

	files, err := fs.ReadDir(os.DirFS(dirname), ".")
	if err != nil {
		fmt.Println(err)
		return err
	}

	for _, file := range files {
		name := file.Name()

		if file.IsDir() {
			name = name + "/"
		}

		fmt.Printf("%s%s\n", indent, name)

		if file.IsDir() && maxDepth > 0 {
			err = readDir(dirname+"/"+file.Name(), maxDepth, indent+"  ")
			if err != nil {
				fmt.Printf("Error reading directory: %s\n", err.Error())
			}
		}
	}

	return nil
}

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List files in a directory",
	Long:  `List files in a directory`,
	Run: func(cmd *cobra.Command, args []string) {
		err := readDir(".", 2, "")
		if err != nil {
			panic(err.Error())
		}
	},
}
