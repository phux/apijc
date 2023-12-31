package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	jd "github.com/josephburnett/jd/lib"
	"golang.org/x/time/rate"
)

var (
	ErrNoTargetsDefined                       = errors.New("no URL targets defined")
	ErrJSONMismatch                           = errors.New("JSON mismatch")
	ErrDomainsMatch                           = errors.New("base and newDomain cannot be the same domain")
	ErrUnexpectedStatusCode                   = errors.New("unexpected status code")
	ErrPrefixFilledButSuffixNot               = errors.New("PatternPrefix is filled but PatternSuffix is not")
	ErrSuffixFilledButPrefixNot               = errors.New("PatternSuffix is filled but PatternPrefix is not")
	ErrBothRequestBodyAndRequestBodyFileGiven = errors.New("must have only one of requestBody or requestBodyFile - both are given")
	ErrRequestBodyFileNotFound                = errors.New("could not find requestBodyFile")
)

type parser interface {
	ParsePath(string, Options) ([]string, error)
}

type limiter interface {
	Wait(context.Context) error
}

type App struct {
	BaseDomain string
	NewDomain  string
	URLs       URLs
	Results    *Results
	parser     parser
	limiter    limiter
	headers    Headers
}

func NewApp(
	baseDomain, newDomain string,
	parser parser,
	rateLimit float64,
	headers Headers,
) *App {
	return &App{
		BaseDomain: baseDomain,
		NewDomain:  newDomain,
		URLs: URLs{
			Targets: []Target{},
		},
		parser:  parser,
		limiter: rate.NewLimiter(rate.Limit(rateLimit), 1),
		headers: headers,
		Results: &Results{
			Findings: []Finding{},
		},
	}
}

func (a *App) Run() error {
	if len(a.URLs.Targets) == 0 && len(a.URLs.SequentialTargets) == 0 {
		return ErrNoTargetsDefined
	}

	if a.BaseDomain == a.NewDomain {
		return ErrDomainsMatch
	}

	totalPaths := 0
	totalCheckedPaths := 0
	for _, target := range a.URLs.Targets {
		log.Printf("Checking %s %s\n", target.HTTPMethod, target.RelativePath)

		var err error
		totalCheckedPaths, totalPaths, err = a.ProcessTarget(target, totalCheckedPaths, totalPaths)
		if err != nil {
			return err
		}
	}

	for name, targets := range a.URLs.SequentialTargets {
		log.Printf("Checking sequential group: %s\n", name)
		for _, target := range targets {
			var err error
			totalCheckedPaths, totalPaths, err = a.ProcessTarget(target, totalCheckedPaths, totalPaths)
			if err != nil {
				return err
			}
		}
	}

	log.Printf(
		"Done. Checked %d of %d paths\n\n",
		totalCheckedPaths,
		totalPaths,
	)

	return nil
}

func (a *App) ProcessTarget(target Target, totalCheckedPaths int, totalPaths int) (int, int, error) {
	initialFindings := 0
	if a.Results != nil {
		initialFindings = len(a.Results.Findings)
	}

	checkedPaths, countPaths, err := a.CheckTarget(target)
	totalCheckedPaths += checkedPaths
	totalPaths += countPaths
	if err != nil {
		log.Println(err)

		return 0, 0, err
	}

	result := "Success"
	if len(a.Results.Findings) != initialFindings {
		result = "ERROR"
	}

	log.Printf(
		"%s: %s %s (checked %d of %d paths)\n\n",
		result,
		target.HTTPMethod,
		target.RelativePath,
		checkedPaths,
		countPaths,
	)
	return totalCheckedPaths, totalPaths, nil
}

func (a *App) buildOptsFromTarget(target Target) (Options, error) {
	err := a.validatePatternPrefixAndSuffixMatch(target)
	if err != nil {
		return Options{}, err
	}

	opts := Options{
		PatternPrefix: "{",
		PatternSuffix: "}",
	}

	if target.PatternPrefix != nil && target.PatternSuffix != nil {
		opts.PatternPrefix = *target.PatternPrefix
		opts.PatternSuffix = *target.PatternSuffix
	}

	return opts, nil
}

func (*App) validatePatternPrefixAndSuffixMatch(target Target) error {
	if target.PatternPrefix != nil && target.PatternSuffix == nil {
		return fmt.Errorf(
			"%s: %w",
			target.RelativePath,
			ErrPrefixFilledButSuffixNot,
		)
	}

	if target.PatternPrefix == nil && target.PatternSuffix != nil {
		return fmt.Errorf(
			"%s: %w",
			target.RelativePath,
			ErrSuffixFilledButPrefixNot,
		)
	}

	return nil
}

func (a *App) CheckTarget(target Target) (int, int, error) {
	opts, err := a.buildOptsFromTarget(target)
	if err != nil {
		return 0, 0, err
	}

	relativePaths, err := a.parser.ParsePath(target.RelativePath, opts)
	countPaths := len(relativePaths)
	checkedPaths := 0
	if err != nil {
		a.addFinding(target.RelativePath, "", err)

		return checkedPaths, countPaths,
			fmt.Errorf(
				"CheckTarget: could not resolve relative paths: %w",
				err,
			)
	}

	ctx := context.Background()
	for _, relativePath := range relativePaths {
		err := a.limiter.Wait(ctx)
		if err != nil {
			return checkedPaths, countPaths,
				fmt.Errorf("error while rate limiting: %w", err)
		}

		baseURL := a.BaseDomain + relativePath
		baseBodyJSON, err := a.callTarget(baseURL, target)
		if err != nil {
			a.addFinding(baseURL, "", err)

			return checkedPaths, countPaths, nil
		}

		newURL := a.NewDomain + relativePath
		newBodyJSON, err := a.callTarget(newURL, target)
		if err != nil {
			a.addFinding(newURL, "", err)

			return checkedPaths, countPaths, nil
		}

		diff := a.compareResponseBodies(baseBodyJSON, newBodyJSON)
		if diff != "" {
			a.addFinding(relativePath, diff, ErrJSONMismatch)
		}

		checkedPaths++
	}

	return checkedPaths, countPaths, nil
}

func (a *App) compareResponseBodies(
	baseBodyJSON, newBodyJSON []byte,
) string {
	first, _ := jd.ReadJsonString(string(baseBodyJSON))
	second, _ := jd.ReadJsonString(string(newBodyJSON))
	diff := first.Diff(second)

	return diff.Render()
}

func (a *App) AddURLs(urls URLs) {
	a.URLs = urls
}

func (a *App) callTarget(url string, target Target) ([]byte, error) {
	res, err := a.makeHTTPRequest(url, target)
	if err != nil {
		return nil, a.requestError(res, target, err)
	}

	if res.StatusCode != target.ExpectedStatusCode {
		return nil, a.statusCodeMissmatchError(target, res)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	return body, nil
}

func (a *App) statusCodeMissmatchError(target Target, res *http.Response) error {
	err := fmt.Errorf(
		"%w: expected %d, got %d",
		ErrUnexpectedStatusCode,
		target.ExpectedStatusCode,
		res.StatusCode,
	)

	return err
}

func (a *App) requestError(res *http.Response, target Target, err error) error {
	statusCode := 0
	if res != nil {
		statusCode = res.StatusCode
	}
	errWrapped := fmt.Errorf(
		"unexpected status code: expected %d, got %d; %w",
		target.ExpectedStatusCode,
		statusCode,
		err,
	)

	return errWrapped
}

func (a *App) makeHTTPRequest(url string, target Target) (*http.Response, error) {
	req, err := a.buildRequest(target, url)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client: error making http request: %w", err)
	}

	return res, nil
}

func (a *App) buildRequest(target Target, url string) (*http.Request, error) {
	var req *http.Request
	var err error

	if target.RequestBody != nil && target.RequestBodyFile != nil {
		return nil, ErrBothRequestBodyAndRequestBodyFileGiven
	}

	if target.RequestBody != nil {
		req, err = http.NewRequest(target.HTTPMethod, url, strings.NewReader(*target.RequestBody))
	}
	if target.RequestBodyFile != nil {
		if _, err := os.Stat(*target.RequestBodyFile); err != nil {
			return nil, fmt.Errorf(
				"%s: %w",
				*target.RequestBodyFile,
				ErrRequestBodyFileNotFound,
			)
		}

		body, err := os.ReadFile(*target.RequestBodyFile)
		if err != nil {
			return nil, err
		}
		req, err = http.NewRequest(target.HTTPMethod, url, strings.NewReader(string(body)))
	}
	if req == nil {
		req, err = http.NewRequest(target.HTTPMethod, url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("client: could not create request: %w", err)
	}

	a.setHeaders(req, url, target)

	return req, nil
}

func (a *App) setHeaders(req *http.Request, url string, target Target) {
	for key, value := range a.headers.Global {
		req.Header.Set(key, value)
	}

	domainSpecificHeaders := a.headers.BaseDomain
	if !a.isBaseDomain(url) {
		domainSpecificHeaders = a.headers.NewDomain
	}
	for key, value := range domainSpecificHeaders {
		req.Header.Set(key, value)
	}

	for key, value := range target.RequestHeaders {
		req.Header.Set(key, value)
	}
}

func (a *App) addFinding(url, diff string, err error) {
	a.Results.Findings = append(
		a.Results.Findings,
		Finding{URL: url, Diff: diff, Error: fmt.Sprint(err)},
	)
}

func (a *App) isBaseDomain(url string) bool {
	return strings.HasPrefix(url, a.BaseDomain)
}
