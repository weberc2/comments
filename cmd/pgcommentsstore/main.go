package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
	"github.com/weberc2/comments/pkg/pgcommentsstore"
	"github.com/weberc2/comments/pkg/types"
)

type Store = pgcommentsstore.PGCommentsStore

func main() {
	app := cli.App{
		Name:        "pgcommentsstore",
		Description: "a command line `PGCommentsStore` intreface",
		Commands: []*cli.Command{{
			Name:        "table",
			Description: "commands for interacting with the backing pg table",
			Subcommands: cli.Commands{{
				Name:        "ensure",
				Aliases:     []string{"create", "make"},
				Description: "create the table if it doesn't already exist",
				Action: withStore(func(store *Store, ctx *cli.Context) error {
					return store.EnsureTable()
				}),
			}, {}, {
				Name:        "drop",
				Aliases:     []string{"delete", "destroy"},
				Description: "drop the postgres table",
				Action: withStore(func(store *Store, ctx *cli.Context) error {
					return store.DropTable()
				}),
			}, {
				Name:        "reset",
				Description: "delete and recreate the postgres table",
				Action: withStore(func(store *Store, ctx *cli.Context) error {
					return store.ResetTable()
				}),
			}, {
				Name: "clear",
				Description: "clear the rows from the table without " +
					"dropping it",
				Action: withStore(func(store *Store, ctx *cli.Context) error {
					return store.ClearTable()
				}),
			}},
		}, {
			Name:        "comments",
			Description: "commands for managing comments",
			Subcommands: []*cli.Command{{
				Name:        "list",
				Description: "list comments",
				Action: withStore(func(store *Store, ctx *cli.Context) error {
					comments, err := store.List()
					if err != nil {
						return err
					}

					data, err := json.MarshalIndent(comments, "", "  ")
					if err != nil {
						return err
					}

					_, err = fmt.Printf("%s\n", data)
					return err
				}),
			}, {
				Name:        "put",
				Aliases:     []string{"add", "create", "make", "insert"},
				Description: "add a comment",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "id",
						DefaultText: "<a new uuid>",
						Usage:       "the comment's ID",
						Required:    false,
					},
					&cli.StringFlag{
						Name:     "post",
						Usage:    "the comment's post",
						Required: true,
					},
					&cli.StringFlag{
						Name:        "parent",
						DefaultText: "<id of the toplevel comment>",
						Usage:       "the comment's parent",
						Required:    false,
					},
					&cli.StringFlag{
						Name:     "author",
						Usage:    "the comment's author",
						Required: true,
					},
					&cli.TimestampFlag{
						Name:        "created",
						DefaultText: "<the current time>",
						Usage:       "the time the comment was created",
						Layout:      time.RFC3339,
						Required:    false,
					},
					&cli.TimestampFlag{
						Name:        "modified",
						DefaultText: "<the current time>",
						Usage:       "the time the comment was created",
						Layout:      time.RFC3339,
						Required:    false,
					},
					&cli.BoolFlag{
						Name:        "deleted",
						DefaultText: "<false>",
						Usage: "whether or not the comment should be " +
							"considered deleted.",
						Required: false,
					},
					&cli.StringFlag{
						Name:        "body",
						DefaultText: "empty",
						Required:    false,
					},
				},
				Action: withStore(func(store *Store, ctx *cli.Context) error {
					var (
						now      = time.Now()
						created  = now
						modified = now
						err      error
					)

					if t := ctx.Timestamp("created"); t != nil {
						created = *t
					}

					if t := ctx.Timestamp("modified"); t != nil {
						modified = *t
					}

					input := &types.Comment{
						ID:       types.CommentID(ctx.String("id")),
						Post:     types.PostID(ctx.String("post")),
						Parent:   types.CommentID(ctx.String("parent")),
						Author:   types.UserID(ctx.String("author")),
						Created:  created,
						Modified: modified,
						Deleted:  ctx.Bool("deleted"),
						Body:     ctx.String("body"),
					}

					if input.ID == "" {
						input.ID = types.CommentID(uuid.NewString())
					}
					if err := store.Put(input); err != nil {
						return err
					}
					data, err := json.MarshalIndent(input, "", "  ")
					if err != nil {
						return err
					}
					_, err = fmt.Printf("%s\n", data)
					return err
				}),
			}, {
				Name:        "update",
				Description: "update a comment",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "id",
						Usage:    "the comment's id",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "post",
						Usage:    "the comment's post",
						Required: true,
					},
					&cli.StringFlag{
						Name:        "parent",
						DefaultText: "<no change>",
						Usage:       "the comment's parent",
						Required:    false,
					},
					&cli.StringFlag{
						Name:        "author",
						DefaultText: "<no change>",
						Usage:       "the comment's author",
						Required:    false,
					},
					&cli.TimestampFlag{
						Name:        "created",
						DefaultText: "<no change>",
						Usage:       "the time the comment was created",
						Layout:      time.RFC3339,
						Required:    false,
					},
					&cli.TimestampFlag{
						Name:        "modified",
						DefaultText: "<no change>",
						Usage:       "the time the comment was created",
						Layout:      time.RFC3339,
						Required:    false,
					},
					&cli.BoolFlag{
						Name:        "deleted",
						DefaultText: "<no change>",
						Usage: "whether or not the comment should be " +
							"considered deleted.",
						Required: false,
					},
					&cli.StringFlag{
						Name:        "body",
						DefaultText: "<no change>",
						Usage:       "The comment's body.",
						Required:    false,
					},
				},
				Action: withStore(func(store *Store, ctx *cli.Context) error {
					cp := types.NewCommentPatch(
						types.CommentID(ctx.String("id")),
						types.PostID(ctx.String("post")),
					)
					if ctx.IsSet("parent") {
						cp.SetParent(types.CommentID(ctx.String("parent")))
					}
					if ctx.IsSet("author") {
						cp.SetAuthor(types.UserID(ctx.String("author")))
					}
					if ctx.IsSet("created") {
						cp.SetCreated(*ctx.Timestamp("created"))
					}
					if ctx.IsSet("modified") {
						cp.SetModified(*ctx.Timestamp("modified"))
					}
					if ctx.IsSet("deleted") {
						cp.SetDeleted(ctx.Bool("deleted"))
					}
					if ctx.IsSet("body") {
						cp.SetBody(ctx.String("body"))
					}

					if err := store.Update(cp); err != nil {
						return err
					}

					data, err := json.MarshalIndent(cp, "", "  ")
					if err != nil {
						return err
					}
					_, err = fmt.Printf("%s\n", data)
					return err
				}),
			}, {
				Name:        "comment",
				Aliases:     []string{"get", "fetch"},
				Description: "Retrieve a comment.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "post",
						Usage:    "the comment's post",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "id",
						Usage:    "the comment's ID",
						Required: true,
					},
				},
				Action: withStore(func(store *Store, ctx *cli.Context) error {
					c, err := store.Comment(
						types.PostID(ctx.String("post")),
						types.CommentID(ctx.String("id")),
					)
					if err != nil {
						return err
					}
					data, err := json.MarshalIndent(c, "", "  ")
					if err != nil {
						return err
					}
					_, err = fmt.Printf("%s\n", data)
					return err
				}),
			}, {
				Name:        "replies",
				Description: "Fetch replies to a comment or post.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "post",
						Usage:    "the comment's post",
						Required: true,
					},
					&cli.StringFlag{
						Name: "parent",
						Usage: "the parent of the replies to be fetched. " +
							"if this is omitted, all comments associated " +
							"with the provided post will be retrieved.",
						Required: false,
					},
				},
				Action: withStore(func(store *Store, ctx *cli.Context) error {
					replies, err := store.Replies(
						types.PostID(ctx.String("post")),
						types.CommentID(ctx.String("parent")),
					)
					if err != nil {
						return err
					}
					data, err := json.MarshalIndent(replies, "", "  ")
					if err != nil {
						return err
					}
					_, err = fmt.Printf("%s\n", data)
					return err
				}),
			}, {
				Name:        "delete",
				Description: "Deletes a comment.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "post",
						Usage:    "the comment's post",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "id",
						Usage:    "the comment's ID",
						Required: true,
					},
					&cli.BoolFlag{
						Name:        "hard",
						DefaultText: "<false>",
						Usage: "completely remove a comment (as opposed to " +
							"tombstoning it)",
						Required: false,
					},
				},
				Action: withStore(func(store *Store, ctx *cli.Context) error {
					p := types.PostID(ctx.String("post"))
					c := types.CommentID(ctx.String("id"))
					if ctx.Bool("hard") {
						return store.Delete(p, c)
					}
					return store.Update(
						types.NewCommentPatch(c, p).SetDeleted(true),
					)
				}),
			}},
		}},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func withStore(
	f func(
		store *Store,
		ctx *cli.Context,
	) error,
) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		store, err := pgcommentsstore.OpenEnv()
		if err != nil {
			log.Fatalf("opening comments store: %v", err)
		}
		return f(store, ctx)
	}
}
