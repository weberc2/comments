package pgcommentsstore

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/weberc2/auth/pkg/pgutil"
	"github.com/weberc2/comments/pkg/types"

	pz "github.com/weberc2/httpeasy"
)

const errUniqueViolation = "23505"

type PGCommentsStore sql.DB

func OpenEnv() (*PGCommentsStore, error) {
	db, err := sql.Open(
		"postgres",
		fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			getEnv("PG_HOST", "localhost"),
			getEnv("PG_PORT", "5432"),
			getEnv("PG_USER", "postgres"),
			getEnv("PG_PASS", ""),
			getEnv("PG_DB_NAME", "postgres"),
			getEnv("PG_SSL_MODE", "disable"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("opening postgres database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging postgres database: %w", err)
	}

	return (*PGCommentsStore)(db), nil
}

func getEnv(env, def string) string {
	x := os.Getenv(env)
	if x == "" {
		return def
	}
	return x
}

func (pgcs *PGCommentsStore) DropTable() error {
	return Table.Drop((*sql.DB)(pgcs))
}

func (pgcs *PGCommentsStore) EnsureTable() error {
	return Table.Ensure((*sql.DB)(pgcs))
}

func (pgcs *PGCommentsStore) ClearTable() error {
	return Table.Clear((*sql.DB)(pgcs))
}

func (pgcs *PGCommentsStore) ResetTable() error {
	return Table.Reset((*sql.DB)(pgcs))
}

func (pgcs *PGCommentsStore) Put(c *types.Comment) error {
	return Table.Insert((*sql.DB)(pgcs), (*comment)(c))
}

func (pgcs *PGCommentsStore) Comment(
	p types.PostID,
	c types.CommentID,
) (*types.Comment, error) {
	var out comment
	if err := Table.Get((*sql.DB)(pgcs), &comment{ID: c}, &out); err != nil {
		return nil, err
	}
	return (*types.Comment)(&out), nil
}

func (pgcs *PGCommentsStore) Replies(
	p types.PostID,
	parent types.CommentID,
) ([]*types.Comment, error) {
	comments, err := pgcs.commentsQuery(
		`WITH RECURSIVE t AS (
	SELECT * FROM comments WHERE post = $1 AND parent = $2 UNION
	SELECT comments.* FROM comments JOIN t ON
	comments.post = t.post AND comments.parent = t.id
) SELECT id, post, parent, author, created, modified, deleted, body FROM t`,
		p,
		parent,
	)
	if err != nil {
		return nil, fmt.Errorf("querying replies from postgres: %w", err)
	}
	return comments, nil
}

func (pgcs *PGCommentsStore) List() ([]*types.Comment, error) {
	result, err := Table.List((*sql.DB)(pgcs))
	if err != nil {
		return nil, fmt.Errorf("listing comments: %w", err)
	}
	var values []comment
	var comments []*types.Comment
	for result.Next() {
		values = append(values, comment{})
		comment := &values[len(values)-1]
		if err := result.Scan(comment); err != nil {
			return nil, fmt.Errorf("scanning comment: %w", err)
		}
		comments = append(comments, (*types.Comment)(comment))
	}
	return comments, nil
}

func (pgcs *PGCommentsStore) commentsQuery(
	query string,
	vs ...interface{},
) ([]*types.Comment, error) {
	rows, err := (*sql.DB)(pgcs).Query(query, vs...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("PGCommentsStore.Replies(): closing sql.Rows: %v", err)
		}
	}()

	// we want to initialize these so that an empty slice serializes to `[]`
	// instead of `null`.
	buf := []types.Comment{}  // put all results in a single allocation
	out := []*types.Comment{} // every item in `out` points into `buf`
	for i := 0; rows.Next(); i++ {
		buf = append(buf, types.Comment{})
		if err := scanComment(&buf[i], rows); err != nil {
			return nil, fmt.Errorf(
				"scanning postgres row into comment: %w",
				err,
			)
		}
		out = append(out, &buf[i])
	}
	return out, nil
}

func scanComment(
	c *types.Comment,
	s interface{ Scan(...interface{}) error },
) error {
	var createdString, modifiedString string
	if err := s.Scan(
		&c.ID,
		&c.Post,
		&c.Parent,
		&c.Author,
		&createdString,
		&modifiedString,
		&c.Deleted,
		&c.Body,
	); err != nil {
		return err
	}
	created, err := time.Parse(time.RFC3339, createdString)
	if err != nil {
		return fmt.Errorf(
			"parsing `created` time from `%s`: %v",
			createdString,
			err,
		)
	}
	modified, err := time.Parse(time.RFC3339, modifiedString)
	if err != nil {
		return fmt.Errorf(
			"parsing `modified` time from `%s`: %v",
			modifiedString,
			err,
		)
	}
	c.Created = created
	c.Modified = modified
	return nil
}

func (pgcs *PGCommentsStore) Update(c *types.CommentPatch) error {
	if !c.IsSet(types.FieldID) {
		return fmt.Errorf(
			"`CommentPatch` is missing required field `%s`",
			types.FieldID,
		)
	}
	if !c.IsSet(types.FieldPost) {
		return fmt.Errorf(
			"`CommentPatch` is missing required field `%s`",
			types.FieldPost,
		)
	}

	columns, params := fieldsToColumnsAndParams(c)
	// The `RETURNING id` is required to provoke a `sql.ErrNoRows` response in
	// cases where the `(post, id)` tuple is not found. Similarly, the `dummy`
	// variable is required to prevent the `Scan()` call from failing.
	var dummy string
	if err := (*sql.DB)(pgcs).QueryRow(
		fmt.Sprintf(
			"UPDATE comments SET %s WHERE id=$1 AND post=$2 RETURNING id",
			columns,
		),
		params...,
	).Scan(&dummy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = &types.CommentNotFoundErr{Post: c.Post(), Comment: c.ID()}
		}
		return fmt.Errorf("updating comment in postgres: %w", err)
	}
	return nil
}

func fieldsToColumnsAndParams(cp *types.CommentPatch) (string, []interface{}) {
	var (
		fields  = cp.Fields()
		params  = []interface{}{cp.ID()}
		columns strings.Builder
		field   types.Field
	)
	columns.WriteString(fieldToColumn(types.Fields[0]))
	columns.WriteString("=$1")
	for _, field = range types.Fields[1:] {
		if fields.Contains(field) {
			params = append(params, fieldToSQLParam(cp, field))
			columns.WriteString(", ")
			columns.WriteString(fieldToColumn(field))
			columns.WriteByte('=')
			columns.WriteByte('$')
			columns.WriteString(strconv.Itoa(len(params)))
		}
	}
	return columns.String(), params
}

func fieldToColumn(field types.Field) string {
	// At some point in the future, column names and field names might differ
	// (e.g., if we add a field with multiple words, we'll camel-case the
	// field name but snake-case the column name since Postgres doesn't do well
	// with case sensitivity).
	return field.String()
}

func fieldToSQLParam(cp *types.CommentPatch, field types.Field) interface{} {
	switch field {
	case types.FieldID:
		return cp.ID()
	case types.FieldPost:
		return cp.Post()
	case types.FieldParent:
		return cp.Parent()
	case types.FieldAuthor:
		return cp.Author()
	case types.FieldCreated:
		return cp.Created().Format(time.RFC3339)
	case types.FieldModified:
		return cp.Modified().Format(time.RFC3339)
	case types.FieldDeleted:
		return cp.Deleted()
	case types.FieldBody:
		return cp.Body()
	default:
		panic(fmt.Sprintf("invalid field: %d", field))
	}
}

func (pgcs *PGCommentsStore) Delete(p types.PostID, c types.CommentID) error {
	return Table.Delete((*sql.DB)(pgcs), &comment{ID: c})
}

// Implement `pgutil.Item` for `types.Comment`.
//
// Since the implementation for a `pgutil.Item` is tightly coupled to the table
// (specifically the number and quantity of columns), we're going to collocate
// the implementation with the column definition/specification rather than
// implementing the interface on `types.Comment` directly.
type comment types.Comment

func (c *comment) Values(values []interface{}) {
	values[0] = c.Post
	values[1] = c.ID
	values[2] = c.Parent
	values[3] = c.Author
	values[4] = c.Created
	values[5] = c.Modified
	values[6] = c.Deleted
	values[7] = c.Body
}

func (c *comment) Scan(pointers []interface{}) {
	pointers[0] = &c.Post
	pointers[1] = &c.ID
	pointers[2] = &c.Parent
	pointers[3] = &c.Author
	pointers[4] = &c.Created
	pointers[5] = &c.Modified
	pointers[6] = &c.Deleted
	pointers[7] = &c.Body
}

type commentPatch types.CommentPatch

func (cp *commentPatch) Values(values []interface{}) {
	patch := (*types.CommentPatch)(cp)
	for i, c := range Table.Columns {
		f, err := types.FieldFromName(c.Name)
		if err != nil {
			panic(
				fmt.Sprintf(
					"expecting field names match column names: %v",
					err,
				),
			)
		}
	}
}

var (
	// fail compilation if `comment` doesn't implement the `pgutil.Item`
	// interface.
	_ pgutil.Item         = &comment{}
	_ types.CommentsStore = &PGCommentsStore{}

	fieldsOrderedByColumn = [types.FieldCount]types.Field{
		types.FieldPost,
		types.FieldID,
		types.FieldParent,
		types.FieldAuthor,
		types.FieldCreated,
		types.FieldModified,
		types.FieldDeleted,
		types.FieldBody,
	}

	errCommentExists = &pz.HTTPError{
		Status:  http.StatusConflict,
		Message: "comment exists",
	}
	errCommentNotFound = &pz.HTTPError{
		Status:  http.StatusNotFound,
		Message: "comment not found",
	}
	Table = pgutil.Table{
		Name: "comments",
		PrimaryKeys: []pgutil.Column{{
			Name: "post",
			Type: "VARCHAR(255)",
		}, {
			Name: "id",
			Type: "VARCHAR(255)",
		}},
		OtherColumns: []pgutil.Column{{
			Name:    "parent",
			Type:    "VARCHAR(255)",
			Default: pgutil.NewString(""),
		}, {
			Name: "author",
			Type: "VARCHAR(255)",
		}, {
			Name:    "created",
			Type:    "TIMESTAMPTZ",
			Default: pgutil.SQL("CURRENT_TIMESTAMP"),
		}, {
			Name:    "modified",
			Type:    "TIMESTAMPTZ",
			Default: pgutil.SQL("CURRENT_TIMESTAMP"),
		}, {
			Name:    "deleted",
			Type:    "BOOLEAN",
			Default: pgutil.NewBoolean(false),
		}, {
			Name: "body",
			Type: "VARCHAR(5096)",
		}},
		ExistsErr:   errCommentExists,
		NotFoundErr: errCommentNotFound,
	}
)
