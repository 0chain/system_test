package client

import (
	"fmt"
	"net/url"
	"strings"
)

type URLBuilder struct {
	formattedURL url.URL
	queries      url.Values
}

func (ub *URLBuilder) SetScheme(scheme string) *URLBuilder {
	ub.formattedURL.Scheme = scheme
	return ub
}

func (ub *URLBuilder) SetHost(host string) *URLBuilder {
	ub.formattedURL.Host = host
	return ub
}

func (ub *URLBuilder) SetPath(path string) *URLBuilder {
	ub.formattedURL.Path = path
	return ub
}

func (ub *URLBuilder) SetPathVariable(name, value string) *URLBuilder {
	ub.formattedURL.Path = strings.Replace(ub.formattedURL.Path, fmt.Sprintf(":%s", name), value, 1)
	return ub
}

func (ub *URLBuilder) AddParams(name, value string) *URLBuilder {
	ub.queries.Set(name, value)
	return ub
}

func (ub *URLBuilder) String() string {
	ub.formattedURL.RawQuery = ub.queries.Encode()
	return ub.formattedURL.String()
}

func NewURLBuilder() *URLBuilder {
	return new(URLBuilder)
}
