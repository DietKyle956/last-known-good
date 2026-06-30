package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var followFlag bool

var logsCmd = &cobra.Command{
	Use:   "logs <id>",
	Short: "View session log output",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		logDir := cmd.Flag("log-dir").Value.String()
		path := filepath.Join(logDir, fmt.Sprintf("session_%s.jsonl", id))

		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("log file for session %s not found", id)
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open log file: %w", err)
		}
		defer f.Close()

		if followFlag {
			return followLog(f)
		}

		_, err = io.Copy(os.Stdout, f)
		return err
	},
}

func followLog(f *os.File) error {
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			return fmt.Errorf("follow log: %w", err)
		}
		fmt.Print(line)
	}
}

func init() {
	cwd, _ := os.Getwd()
	defaultLogDir := filepath.Join(cwd, "logs")

	logsCmd.Flags().BoolVarP(&followFlag, "follow", "f", false, "Follow log output in real time")
	logsCmd.Flags().String("log-dir", defaultLogDir, "Directory containing session log files")
	rootCmd.AddCommand(logsCmd)
}
