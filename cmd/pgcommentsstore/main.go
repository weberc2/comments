package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/urfave/cli"
	"github.com/weberc2/comments/pkg/pgcommentsstore"
	"github.com/weberc2/comments/pkg/types"
)

func main() {
	app := cli.App{
		Name: "pgcommentsstore",
		Commands: []cli.Command{
			{
				Name: "table",
				Subcommands: cli.Commands{
					{
						Name:        "ensure",
						Description: "ensure that the `comments` table exists",
						ShortName:   "e",
						Action: withStore(func(
							store *pgcommentsstore.PGCommentsStore,
							ctx *cli.Context,
						) error {
							return store.EnsureTable()
						}),
					},
					{
						Name:        "drop",
						Description: "drop the `comments` table",
						ShortName:   "d",
						Action: withStore(func(
							store *pgcommentsstore.PGCommentsStore,
							ctx *cli.Context,
						) error {
							return store.DropTable()
						}),
					},
					{
						Name:        "reset",
						Description: "reset the `comments` table (drop + ensure)",
						ShortName:   "r",
						Action: withStore(func(
							store *pgcommentsstore.PGCommentsStore,
							ctx *cli.Context,
						) error {
							return store.ResetTable()
						}),
					},
					{
						Name:        "clear",
						Description: "clear the `comments` table",
						ShortName:   "c",
						Action: withStore(func(
							store *pgcommentsstore.PGCommentsStore,
							ctx *cli.Context,
						) error {
							return store.ClearTable()
						}),
					},
				},
			},
			{
				Name:        "list",
				Description: "list comments",
				Action: withStore(func(
					store *pgcommentsstore.PGCommentsStore,
					ctx *cli.Context,
				) error {
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
			},
			{
				Name:        "put",
				Aliases:     []string{"add", "create"},
				Description: "add a comment",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "id",
						Usage:    "The comment's ID. Defaults to a UUID.",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "post",
						Usage:    "The comment's post. Required.",
						Required: true,
					},
					&cli.StringFlag{
						Name: "parent",
						Usage: "The comment's parent. Defaults to " +
							"toplevel comment.",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "author",
						Usage:    "The comment's author. Required.",
						Required: true,
					},
					&cli.StringFlag{
						Name: "created",
						Usage: "The time the comment was created. Defaults " +
							"to the current time.",
						Required: false,
					},
					&cli.StringFlag{
						Name: "modified",
						Usage: "The time the comment was created. Defaults " +
							"to the current time.",
						Required: false,
					},
					&cli.BoolFlag{
						Name: "deleted",
						Usage: "Whether or not the comment should be " +
							"considered deleted. Defaults to `false`.",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "body",
						Usage:    "The comment's body. Defaults to empty.",
						Required: false,
					},
				},
				Action: withStore(func(
					store *pgcommentsstore.PGCommentsStore,
					ctx *cli.Context,
				) error {
					var (
						now      = time.Now()
						created  = now
						modified = now
						err      error
					)

					if s := ctx.String("created"); s != "" {
						created, err = time.Parse(time.RFC3339, s)
					}
					if err != nil {
						return fmt.Errorf("parsing `--created`: %w", err)
					}

					if s := ctx.String("modified"); s != "" {
						modified, err = time.Parse(time.RFC3339, s)
					}
					if err != nil {
						return fmt.Errorf("parsing `--modified`: %w", err)
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
					createdValue := ctx.String("created")
					if createdValue != "" {
						t, err := time.Parse(time.RFC3339, createdValue)
						if err != nil {
							log.Fatalf("parsing `created` flag: %v", err)
						}
						input.Created = t
					} else {
						input.Created = time.Now().UTC()
					}
					modifiedValue := ctx.String("modified")
					if modifiedValue != "" {
						t, err := time.Parse(time.RFC3339, modifiedValue)
						if err != nil {
							log.Fatalf("parsing `modified` flag: %v", err)
						}
						input.Modified = t
					} else {
						input.Modified = time.Now().UTC()
					}

					if input.ID == "" {
						input.ID = types.CommentID(uuid.NewString())
					}
					comment, err := store.Put(input)
					if err != nil {
						return err
					}
					data, err := json.MarshalIndent(comment, "", "  ")
					if err != nil {
						return err
					}
					_, err = fmt.Printf("%s\n", data)
					return err
				}),
			},
			{
				Name:        "update",
				Description: "update a comment",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "id",
						Usage:    "The comment's ID. Required",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "post",
						Usage:    "The comment's post. Required.",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "parent",
						Usage:    "The comment's parent.",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "author",
						Usage:    "The comment's author.",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "created",
						Usage:    "The time the comment was created.",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "modified",
						Usage:    "The time the comment was created",
						Required: false,
					},
					&cli.BoolFlag{
						Name: "deleted",
						Usage: "Whether or not the comment should be " +
							"considered deleted.",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "body",
						Usage:    "The comment's body.",
						Required: false,
					},
				},
				Action: withStore(func(
					store *pgcommentsstore.PGCommentsStore,
					ctx *cli.Context,
				) error {
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
						created, err := time.Parse(
							time.RFC3339,
							ctx.String("created"),
						)
						if err != nil {
							return fmt.Errorf("parsing `--created`: %w", err)
						}
						cp.SetCreated(created)
					}
					if ctx.IsSet("modified") {
						modified, err := time.Parse(
							time.RFC3339,
							ctx.String("modified"),
						)
						if err != nil {
							return fmt.Errorf("parsing `--modified`: %w", err)
						}
						cp.SetModified(modified)
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
			},
			{
				Name:        "comment",
				Aliases:     []string{"get", "fetch"},
				Description: "Retrieve a comment.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "post",
						Usage:    "The comment's post.",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "id",
						Usage:    "The comment's ID.",
						Required: true,
					},
				},
				Action: withStore(func(
					store *pgcommentsstore.PGCommentsStore,
					ctx *cli.Context,
				) error {
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
			},
			{
				Name:        "replies",
				Description: "Fetch replies to a comment or post.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "post",
						Usage:    "The comment's post.",
						Required: true,
					},
					&cli.StringFlag{
						Name: "parent",
						Usage: "The parent of the replies to be fetched. " +
							"If this is omitted, all comments associated " +
							"with the provided post will be retrieved.",
						Required: false,
					},
				},
				Action: withStore(func(
					store *pgcommentsstore.PGCommentsStore,
					ctx *cli.Context,
				) error {
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
			},
			{
				Name:        "delete",
				Description: "Deletes a comment.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "post",
						Usage:    "The comment's post.",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "id",
						Usage:    "The comment's ID.",
						Required: true,
					},
					&cli.BoolFlag{
						Name: "hard",
						Usage: "Completely remove a comment (as opposed to " +
							"marking its `Deleted` field). Defaults to " +
							"`false`.",
						Required: false,
					},
				},
				Action: withStore(func(
					store *pgcommentsstore.PGCommentsStore,
					ctx *cli.Context,
				) error {
					p := types.PostID(ctx.String("post"))
					c := types.CommentID(ctx.String("id"))
					if ctx.Bool("hard") {
						return store.Delete(p, c)
					}
					return store.Update(
						types.NewCommentPatch(c, p).SetDeleted(true),
					)
				}),
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func withStore(
	f func(
		store *pgcommentsstore.PGCommentsStore,
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
