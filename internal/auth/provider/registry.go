package provider

import (
	"fmt"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type Registry map[string]OAuthProvider

func (r Registry) Get(name string) (OAuthProvider, error) {
	p, ok := r[name]
	if !ok {
		return nil, fmt.Errorf("%w: unknown oauth provider %q", domain.ErrInvalidInput, name)
	}
	return p, nil
}

func (r Registry) Names() []string {
	names := make([]string, 0, len(r))
	for k := range r {
		names = append(names, k)
	}
	return names
}
