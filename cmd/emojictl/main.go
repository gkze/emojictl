package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/gkze/emojictl/pkg/emojictl"
	"github.com/spf13/cobra"
)

func main() {
	sec, err := emojictl.NewSlackEmojictl(
		os.Getenv("SLACK_WORKSPACE"),
		os.Getenv("SLACK_HEADER_TOKEN"),
		os.Getenv("SLACK_BODY_TOKEN"),
	)
	if err != nil {
		panic(err)
	}

	root := &cobra.Command{
		Use:   "emojictl",
		Short: "Emoji control center",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.AddCommand(
		&cobra.Command{
			Use:           "list",
			Aliases:       []string{"l", "ls"},
			Short:         "List emojis",
			SilenceErrors: true,
			SilenceUsage:  true,
			RunE: func(cmd *cobra.Command, args []string) error {
				emojis, err := sec.ListEmojis(cmd.Context())
				if err != nil {
					return err
				}

				tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
				if _, err := tw.Write([]byte("NAME\tURL\n")); err != nil {
					return err
				}

				for _, emoji := range emojis {
					if _, err := tw.Write([]byte(fmt.Sprintf(
						"%s\t%s\n", emoji.Name, emoji.Location.String(),
					))); err != nil {
						return err
					}
				}

				return tw.Flush()
			},
		},
		&cobra.Command{
			Use:           "add",
			Aliases:       []string{"a"},
			Short:         "Upload emoji",
			SilenceErrors: true,
			SilenceUsage:  true,
			Args:          cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				var u *url.URL

				if strings.Contains(args[0], "://") {
					u, err = url.Parse(args[0])
					if err != nil {
						return err
					}

				} else {
					u = &url.URL{Scheme: "file", Path: args[0]}
				}

				name := emojictl.FilenameNoExt(u.Path)
				err := sec.AddEmoji(cmd.Context(), &emojictl.Emoji{
					Name: emojictl.FilenameNoExt(u.Path), Location: u,
				})
				if err != nil {
					return err
				}

				_, err = fmt.Printf("Added %s as %s\n", u.String(), name)
				return err
			},
		},
		&cobra.Command{
			Use:           "remove",
			Aliases:       []string{"r", "rm"},
			Short:         "Remove emoji",
			SilenceErrors: true,
			SilenceUsage:  true,
			Args:          cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				err := sec.RemoveEmoji(cmd.Context(), &emojictl.Emoji{Name: args[0]})
				if err != nil {
					return err
				}

				_, err = fmt.Printf("Removed %s\n", args[0])
				return err
			},
		},
		&cobra.Command{
			Use:           "alias",
			Aliases:       []string{"a", "al"},
			Short:         "Alias emoji",
			SilenceErrors: true,
			SilenceUsage:  true,
			Args:          cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				err := sec.AliasEmoji(cmd.Context(), args[0], args[1])
				if err != nil {
					return err
				}

				_, err = fmt.Printf("Aliased %s to %s\n", args[0], args[1])
				return err
			},
		},
	)

	if err := root.ExecuteContext(context.Background()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
