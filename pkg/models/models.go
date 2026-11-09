package models

import (
	"context"
	"encoding/json"
	"loyalty/pkg/constants"

	"strings"
	"time"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BaseDbModel struct {
	Trash     bool                `bson:"trash,omitempty" json:"trash,omitempty"`
	CreatedAt time.Time           `bson:"createdAt,omitempty" json:"createdAt,omitempty"`
	UpdatedAt time.Time           `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
	CreatedBy *primitive.ObjectID `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	UpdatedBy *primitive.ObjectID `bson:"updatedBy,omitempty" json:"updatedBy,omitempty"`
}

func (b *BaseDbModel) SetCreatedBy(id primitive.ObjectID) {
	b.CreatedBy = lo.ToPtr(id)
	b.UpdatedBy = lo.ToPtr(id)
	b.CreatedAt = time.Now()
	b.UpdatedAt = time.Now()
}

func (b *BaseDbModel) SetUpdatedBy(id primitive.ObjectID) {
	b.UpdatedBy = lo.ToPtr(id)
	b.UpdatedAt = time.Now()
}

func (b *BaseDbModel) SetTrash(trash bool) {
	b.Trash = trash
}

type File struct {
	Id           primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Path         string             `bson:"path,omitempty" json:"path,omitempty"`
	OriginalName string             `bson:"original_name,omitempty" json:"original_name,omitempty"`
	Expire       time.Time          `bson:"expire,omitempty" json:"expire,omitempty"`
}

type LocalizableString struct {
	En string `bson:"en" json:"en"`
	Ar string `bson:"ar,omitempty" json:"ar,omitempty"` // omitempty to allow empty strings
}

type locStr LocalizableString

func (ls LocalizableString) MarshalJSON(ctx context.Context) ([]byte, error) {
	if ctx == nil {
		return json.Marshal(locStr{ls.En, ls.Ar})
	}
	lang := ctx.Value(constants.CtxLangKey).(string)
	if lang == "" {
		return json.Marshal(locStr{ls.En, ls.Ar})
	}
	return []byte(`"` + ls.Localize(lang) + `"`), nil

}

func (ls LocalizableString) Localize(languageKey string) string {
	switch languageKey {
	case "ar":
		if ls.Ar != "" {
			return strings.TrimSpace(ls.Ar)
		}
	}
	return strings.TrimSpace(ls.En)
}
