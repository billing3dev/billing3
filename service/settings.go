package service

import (
	"billing3/database"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var (
	SettingSiteName         = newSetting("site_name", "billing3", true)
	SettingTurnstileSiteKey = newSetting("cf_turnstile_site_key", "", true)
	SettingTurnstileSecret  = newSetting("cf_turnstile_secret", "", false)
	SettingIndexMarkdown    = newSetting("index_markdown", "# Welcome to billing3", true)

	Settings = []Setting{
		SettingSiteName,
		SettingTurnstileSiteKey,
		SettingTurnstileSecret,
		SettingIndexMarkdown,
	}
)

func newSetting(key, defaultValue string, public bool) Setting {
	return Setting{
		key:          key,
		defaultValue: defaultValue,
		public:       public,
	}
}

type Setting struct {
	key          string
	defaultValue string
	public       bool
}

func (s Setting) Key() string {
	return s.key
}

func (s Setting) IsPublic() bool {
	return s.public
}

func (s Setting) Get(ctx context.Context) string {
	ss, err := database.Q.FindSettingByKey(ctx, s.key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.defaultValue
		}
		panic(fmt.Errorf("get setting \"%s\": %w", s.key, err))
	}

	return ss.Value
}

func (s Setting) Set(ctx context.Context, value string) {
	err := database.Q.UpdateSetting(ctx, database.UpdateSettingParams{
		Key:   s.key,
		Value: value,
	})
	if err != nil {
		panic(fmt.Errorf("set setting \"%s\": %w", s.key, err))
	}
}
