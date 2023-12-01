package app

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	valid "github.com/asaskevich/govalidator"
)

var (
	errParserInvalidAmountOfRangeParts = errors.New("invalid number range")
	errParserInvalidRangeType          = errors.New("not a valid number range")
)

type Parser struct{}

func NewURLParser() Parser {
	return Parser{}
}

type Options struct {
	PatternPrefix string // default: {
	PatternSuffix string // default: }
}

func (p Parser) ParsePath(path string, opts Options) ([]string, error) {
	if opts.PatternPrefix == "" || opts.PatternSuffix == "" {
		return []string{}, fmt.Errorf(
			"ParsePath: %+v: PatternPrefix and PatternSuffix cannot be empty",
			opts,
		)
	}

	pattern := regexp.MustCompile(opts.PatternPrefix + "([a-zA-Z0-9,-.]+)" + opts.PatternSuffix)
	if !pattern.MatchString(path) {
		return []string{path}, nil
	}

	targets := []string{}
	matches := pattern.FindAllStringSubmatch(path, -1)
	match := matches[0]
	parts := strings.Split(match[1], ",")
	for _, part := range parts {
		if !strings.Contains(part, "-") {
			if len(matches) == 1 {
				targets = append(
					targets,
					strings.Replace(path, match[0], part, 1),
				)
			} else {
				subtargets, err := p.ParsePath(
					strings.Replace(path, match[0], part, 1),
					opts,
				)
				if err != nil {
					return nil, err
				}

				targets = append(
					targets,
					subtargets...,
				)
			}

			continue
		}

		rangeParts := strings.Split(part, "-")
		rangeParts, err := p.validateRangeParts(rangeParts, part)
		if err != nil {
			return []string{}, err
		}

		first, _ := strconv.ParseInt(rangeParts[0], 10, 64)
		second, _ := strconv.ParseInt(rangeParts[1], 10, 64)
		for i := first; i <= second; i++ {
			targets = append(
				targets,
				strings.Replace(path, match[0], strconv.FormatInt(i, 10), 1),
			)
		}
	}

	return targets, nil
}

func (Parser) validateRangeParts(rangeParts []string, part string) ([]string, error) {
	// sanitize negative numbers
	if len(rangeParts) > 2 {
		if rangeParts[0] == "" {
			rangeParts = append([]string{"-" + rangeParts[1]}, rangeParts[2:]...)
		}
	}
	if len(rangeParts) > 2 {
		if rangeParts[1] == "" {
			rangeParts = []string{rangeParts[0], "-" + rangeParts[2]}
		}
	}

	if len(rangeParts) != 2 {
		return nil, fmt.Errorf("%q: number of elements != 2, is %d: %w", part, len(rangeParts), errParserInvalidAmountOfRangeParts)
	}

	if !valid.IsInt(rangeParts[0]) {
		return nil, fmt.Errorf("%q: first number: %w", part, errParserInvalidRangeType)
	}
	if !valid.IsInt(rangeParts[1]) {
		return nil, fmt.Errorf("%q: second number: %w", part, errParserInvalidRangeType)
	}

	first, _ := strconv.ParseInt(rangeParts[0], 10, 64)
	second, _ := strconv.ParseInt(rangeParts[1], 10, 64)
	if second < first {
		return nil, fmt.Errorf("%q: first number cannot be bigger than second number: %w", part, errParserInvalidRangeType)
	}

	return rangeParts, nil
}
