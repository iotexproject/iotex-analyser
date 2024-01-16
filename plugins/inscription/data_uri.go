package main

import (
	"errors"
	"regexp"
)

var dataURIRegexp = regexp.MustCompile(`data:(?P<mediatype>(?P<mimetype>.+?/.+?)?(?P<parameters>(?:;.+?=.+?)*)?)(?P<extension>;base64)?,(?P<data>.*)`)

// DataURI represents a data URI
type DataURI struct {
	MIMEType   string
	Parameters string
	Extension  string
	Data       string
}

// ParseDataURI parses a data URI
func ParseDataURI(uri string) (DataURI, error) {
	match := dataURIRegexp.FindStringSubmatch(uri)
	if match == nil {
		return DataURI{}, errors.New("invalid data uri")
	}

	return DataURI{
		MIMEType:   match[dataURIRegexp.SubexpIndex("mimetype")],
		Parameters: match[dataURIRegexp.SubexpIndex("parameters")],
		Extension:  match[dataURIRegexp.SubexpIndex("extension")],
		Data:       match[dataURIRegexp.SubexpIndex("data")],
	}, nil
}
