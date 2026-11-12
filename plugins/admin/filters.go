package admin

import (
	"errors"
	"strconv"
	"time"

	"github.com/thecodearcher/limen"
)

// UserFilterFunc turns a filter value into conditions on the users table. The first
// condition is ANDed with the rest of the query. Starting with .Or() joins the previous group.
// Errors are reported as "filter[name] <error>", so phrase them like "must be true or false".
type UserFilterFunc func(value string) ([]limen.Where, error)

// EqualFilter matches column against the value, for string columns.
func EqualFilter(column string) UserFilterFunc {
	return func(value string) ([]limen.Where, error) {
		return []limen.Where{limen.Eq(column, value)}, nil
	}
}

func (p *adminPlugin) builtinUserFilters() map[string]UserFilterFunc {
	filters := map[string]UserFilterFunc{
		"email_verified": p.emailVerifiedFilter,
		"created_after":  p.createdAfterFilter,
		"created_before": p.createdBeforeFilter,
	}
	if p.config.bansEnabled {
		filters["banned"] = p.bannedFilter
	}
	return filters
}

func (p *adminPlugin) emailVerifiedFilter(value string) ([]limen.Where, error) {
	verified, err := strconv.ParseBool(value)
	if err != nil {
		return nil, errors.New("must be true or false")
	}

	column := p.core.Schema.User.GetEmailVerifiedAtField()
	if verified {
		return []limen.Where{limen.IsNotNull(column)}, nil
	}
	return []limen.Where{limen.IsNull(column)}, nil
}

func (p *adminPlugin) createdAfterFilter(value string) ([]limen.Where, error) {
	at, _, err := parseFilterTime(value)
	if err != nil {
		return nil, err
	}
	return []limen.Where{limen.Gte(p.core.Schema.User.GetField(limen.SchemaCreatedAtField), at)}, nil
}

func (p *adminPlugin) createdBeforeFilter(value string) ([]limen.Where, error) {
	at, dateOnly, err := parseFilterTime(value)
	if err != nil {
		return nil, err
	}

	column := p.core.Schema.User.GetField(limen.SchemaCreatedAtField)
	if dateOnly {
		return []limen.Where{limen.Lt(column, at.AddDate(0, 0, 1))}, nil
	}
	return []limen.Where{limen.Lte(column, at)}, nil
}

func parseFilterTime(value string) (at time.Time, dateOnly bool, err error) {
	if at, err := time.Parse(time.RFC3339, value); err == nil {
		return at, false, nil
	}
	day, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Time{}, false, errors.New("must be a date (2006-01-02) or an RFC 3339 time")
	}
	return day, true, nil
}

func (p *adminPlugin) bannedFilter(value string) ([]limen.Where, error) {
	banned, err := strconv.ParseBool(value)
	if err != nil {
		return nil, errors.New("must be true or false")
	}

	bannedAt, expiresAt, now := p.userSchema.GetBannedAtField(), p.userSchema.GetBanExpiresAtField(), time.Now()
	if banned {
		return []limen.Where{
			limen.IsNotNull(bannedAt),
			limen.IsNull(expiresAt),
			limen.Gt(expiresAt, now).Or(),
		}, nil
	}
	return []limen.Where{
		limen.IsNull(bannedAt),
		limen.Lte(expiresAt, now).Or(),
	}, nil
}

func (p *adminPlugin) isBanned(user *limen.User, now time.Time) bool {
	raw := user.Raw()
	if raw[p.userSchema.GetBannedAtField()] == nil {
		return false
	}
	expiresAt := limen.GetNullableValue[time.Time](raw[p.userSchema.GetBanExpiresAtField()])
	return expiresAt == nil || expiresAt.After(now)
}
