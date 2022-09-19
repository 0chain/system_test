package client

import (
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"strings"
)

type URLBuilder struct {
	formattedURL, shiftedURL url.URL
	queries                  url.Values
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
	ub.formattedURL.Path = filepath.Join(ub.formattedURL.Path, path)
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

//
func (ub *URLBuilder) MustShiftParse(rawURL string) *URLBuilder {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		log.Fatalln(err)
	}

	ub.shiftedURL = ub.formattedURL
	ub.formattedURL = *parsedURL

	ub.SetPath(ub.shiftedURL.Path)

	return ub
}

func (ub *URLBuilder) String() string {
	defer func() {
		ub.formattedURL = ub.shiftedURL
	}()
	ub.formattedURL.RawQuery = ub.queries.Encode()
	return ub.formattedURL.String()
}

func NewURLBuilder() *URLBuilder {
	return &URLBuilder{queries: make(url.Values)}
}
