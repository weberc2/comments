package types

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type Field int

const (
	FieldID Field = 1 << iota
	FieldPost
	FieldParent
	FieldAuthor
	FieldCreated
	FieldModified
	FieldDeleted
	FieldBody
)

var Fields = []Field{
	FieldID,
	FieldPost,
	FieldParent,
	FieldAuthor,
	FieldCreated,
	FieldModified,
	FieldDeleted,
	FieldBody,
}

type FieldMask int

func (f Field) Mask() FieldMask { return FieldMask(f) }

func (mask FieldMask) Contains(field Field) bool {
	return field&Field(mask) == field
}

func (mask *FieldMask) Push(field Field) { *mask |= field.Mask() }

func FieldFromName(name string) (Field, bool) {
	switch name {
	case "id":
		return FieldID, true
	case "post":
		return FieldPost, true
	case "parent":
		return FieldParent, true
	case "author":
		return FieldAuthor, true
	case "created":
		return FieldCreated, true
	case "modified":
		return FieldModified, true
	case "deleted":
		return FieldDeleted, true
	case "body":
		return FieldBody, true
	default:
		return 0, false
	}
}

func (field Field) String() string {
	switch field {
	case FieldID:
		return "id"
	case FieldPost:
		return "post"
	case FieldParent:
		return "parent"
	case FieldAuthor:
		return "author"
	case FieldCreated:
		return "created"
	case FieldModified:
		return "modified"
	case FieldDeleted:
		return "deleted"
	case FieldBody:
		return "body"
	default:
		panic(fmt.Sprintf("invalid field: %d", field))
	}
}

func (field Field) GoString() string {
	switch field {
	case FieldID:
		return "ID"
	case FieldPost:
		return "Post"
	case FieldParent:
		return "Parent"
	case FieldAuthor:
		return "Author"
	case FieldCreated:
		return "Created"
	case FieldModified:
		return "Modified"
	case FieldDeleted:
		return "Deleted"
	case FieldBody:
		return "Body"
	default:
		panic(fmt.Sprintf("invalid field: %d", field))
	}
}

type CommentPatch struct {
	comment Comment
	fields  FieldMask
}

func NewCommentPatch(id CommentID, post PostID) *CommentPatch {
	return (&CommentPatch{}).SetID(id).SetPost(post)
}

func (cp *CommentPatch) ID() CommentID       { return cp.comment.ID }
func (cp *CommentPatch) Post() PostID        { return cp.comment.Post }
func (cp *CommentPatch) Parent() CommentID   { return cp.comment.Parent }
func (cp *CommentPatch) Author() UserID      { return cp.comment.Author }
func (cp *CommentPatch) Created() time.Time  { return cp.comment.Created }
func (cp *CommentPatch) Modified() time.Time { return cp.comment.Modified }
func (cp *CommentPatch) Deleted() bool       { return cp.comment.Deleted }
func (cp *CommentPatch) Body() string        { return cp.comment.Body }

func (cp *CommentPatch) SetID(id CommentID) *CommentPatch {
	cp.comment.ID = id
	cp.fields.Push(FieldID)
	return cp
}

func (cp *CommentPatch) SetPost(post PostID) *CommentPatch {
	cp.comment.Post = post
	cp.fields.Push(FieldPost)
	return cp
}

func (cp *CommentPatch) SetParent(parent CommentID) *CommentPatch {
	cp.comment.Parent = parent
	cp.fields.Push(FieldParent)
	return cp
}

func (cp *CommentPatch) SetAuthor(author UserID) *CommentPatch {
	cp.comment.Author = author
	cp.fields.Push(FieldAuthor)
	return cp
}

func (cp *CommentPatch) SetCreated(created time.Time) *CommentPatch {
	cp.comment.Created = created
	cp.fields.Push(FieldCreated)
	return cp
}

func (cp *CommentPatch) SetModified(modified time.Time) *CommentPatch {
	cp.comment.Modified = modified
	cp.fields.Push(FieldModified)
	return cp
}

func (cp *CommentPatch) SetDeleted(deleted bool) *CommentPatch {
	cp.comment.Deleted = deleted
	cp.fields.Push(FieldDeleted)
	return cp
}

func (cp *CommentPatch) SetBody(body string) *CommentPatch {
	cp.comment.Body = body
	cp.fields.Push(FieldBody)
	return cp
}

func (cp *CommentPatch) IsSet(field Field) bool {
	return cp.fields.Contains(field)
}

func (cp *CommentPatch) MarshalJSON() ([]byte, error) {
	return cp.comment.marshalFields(cp.fields)
}

func (cp *CommentPatch) Fields() FieldMask { return cp.fields }

func (c *Comment) marshalFields(fields FieldMask) ([]byte, error) {
	out := map[string]json.RawMessage{}
	for _, field := range Fields {
		if fields.Contains(field) {
			fieldName := field.String()
			data, err := c.marshalField(field)
			if err != nil {
				return nil, fmt.Errorf("marshaling field: %s", fieldName)
			}
			out[fieldName] = json.RawMessage(data)
		}
	}
	return json.Marshal(out)
}

func (c *Comment) marshalField(field Field) ([]byte, error) {
	switch field {
	case FieldID:
		return json.Marshal(&c.ID)
	case FieldPost:
		return json.Marshal(&c.Post)
	case FieldParent:
		return json.Marshal(&c.Parent)
	case FieldAuthor:
		return json.Marshal(&c.Author)
	case FieldCreated:
		return json.Marshal(&c.Created)
	case FieldModified:
		return json.Marshal(&c.Modified)
	case FieldDeleted:
		return json.Marshal(&c.Deleted)
	case FieldBody:
		return json.Marshal(&c.Body)
	default:
		panic(fmt.Sprintf("invalid field: %d", field))
	}
}

func (cp *CommentPatch) UnmarshalJSON(data []byte) error {
	var tmp CommentPatch
	if err := unmarshalCommentPatch(&tmp, data); err != nil {
		return fmt.Errorf("unmarshaling `CommentPatch`: %w", err)
	}
	*cp = tmp
	return nil
}

func (c *Comment) unmarshalField(field Field, data []byte) error {
	switch field {
	case FieldID:
		return json.Unmarshal(data, &c.ID)
	case FieldPost:
		return json.Unmarshal(data, &c.Post)
	case FieldParent:
		return json.Unmarshal(data, &c.Parent)
	case FieldAuthor:
		return json.Unmarshal(data, &c.Author)
	case FieldCreated:
		return json.Unmarshal(data, &c.Created)
	case FieldModified:
		return json.Unmarshal(data, &c.Modified)
	case FieldDeleted:
		return json.Unmarshal(data, &c.Deleted)
	default:
		panic(fmt.Sprintf("invalid field: %d", field))
	}
}

func unmarshalCommentPatch(cp *CommentPatch, data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	for fieldName, message := range fields {
		field, ok := FieldFromName(fieldName)
		if !ok {
			// ignore unknown fields
			log.Printf(
				"ignoring unknown field: %s",
				fieldName,
			)
			continue
		}
		if err := cp.comment.unmarshalField(
			field,
			[]byte(message),
		); err != nil {
			return fmt.Errorf(
				"field `%s`: %v",
				field,
				err,
			)
		}
		cp.fields.Push(field)
	}
	return nil
}

func (cp *CommentPatch) Apply(c *Comment) {
	if cp.IsSet(FieldID) {
		c.ID = cp.ID()
	}
	if cp.IsSet(FieldPost) {
		c.Post = cp.Post()
	}
	if cp.IsSet(FieldParent) {
		c.Parent = cp.Parent()
	}
	if cp.IsSet(FieldAuthor) {
		c.Author = cp.Author()
	}
	if cp.IsSet(FieldCreated) {
		c.Created = cp.Created()
	}
	if cp.IsSet(FieldModified) {
		c.Modified = cp.Modified()
	}
	if cp.IsSet(FieldDeleted) {
		c.Deleted = cp.Deleted()
	}
	if cp.IsSet(FieldBody) {
		c.Body = cp.Body()
	}
}
