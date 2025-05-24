// main.go

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/olebedev/when"
	enRules "github.com/olebedev/when/rules/en"
)

/* ----- data types ----- */

type TimeResult struct {
	Timezone string `json:"timezone"`
	Datetime string `json:"datetime"`
	IsDST    bool   `json:"is_dst"`
}

type TimeConversionResult struct {
	Source         TimeResult `json:"source"`
	Target         TimeResult `json:"target"`
	TimeDifference string     `json:"time_difference"`
}

/* ----- server ----- */

const (
	version = "0.3.2"
	appName = "Time MCP Server"
)

type TimeServer struct {
	localTZ string
	parser  *when.Parser
	nowFunc func() time.Time // New field for injectable "now"
}

// NewTimeServer is the constructor for TimeServer
func NewTimeServer(local string) *TimeServer {
	if local == "" {
		local = detectLocalTZ()
	}
	p := when.New(nil)
	p.Add(enRules.All...) // enable English rules

	return &TimeServer{
		localTZ: local,
		parser:  p,
		nowFunc: time.Now, // Default to actual time.Now
	}
}

// forTesting_SetNowFunc allows tests to override the time.Now() behavior.
// This function is not exported, but can be called from tests in the same package.
func (t *TimeServer) forTesting_SetNowFunc(nowFunc func() time.Time) {
	t.nowFunc = nowFunc
}

/* ----- helpers ----- */

func detectLocalTZ() string {
	// ... (rest of the function is unchanged)
	name, off := time.Now().Zone()
	if name != "" && name != "Local" {
		return name
	}
	h := off / 3600
	m := (off % 3600) / 60
	if m == 0 {
		return fmt.Sprintf("UTC%+d", h)
	}
	return fmt.Sprintf("UTC%+d:%02d", h, m)
}

func atoiStrict(s string) (int, error) {
	// ... (unchanged)
	var v int
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}

/* ----- core methods ----- */

// GetCurrentTime uses the injectable nowFunc
func (t *TimeServer) GetCurrentTime(tz string) (TimeResult, error) {
	if tz == "" {
		tz = t.localTZ
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return TimeResult{}, err
	}
	// Use the injectable nowFunc
	now := t.nowFunc().In(loc)
	return TimeResult{Timezone: tz, Datetime: now.Format(time.RFC3339), IsDST: now.IsDST()}, nil
}

// ConvertTime uses the injectable nowFunc for its date context
func (t *TimeServer) ConvertTime(srcTZ, hhmm, dstTZ string) (TimeConversionResult, error) {
	if srcTZ == "" {
		srcTZ = t.localTZ
	}
	if dstTZ == "" {
		dstTZ = t.localTZ
	}

	srcLoc, err := time.LoadLocation(srcTZ)
	if err != nil {
		return TimeConversionResult{}, err
	}
	dstLoc, err := time.LoadLocation(dstTZ)
	if err != nil {
		return TimeConversionResult{}, err
	}

	parts := strings.Split(hhmm, ":")
	if len(parts) != 2 {
		return TimeConversionResult{}, fmt.Errorf("time must be HH:MM")
	}
	h, errH := atoiStrict(parts[0])
	if errH != nil || h < 0 || h > 23 {
		return TimeConversionResult{}, fmt.Errorf("invalid hour: %s", parts[0])
	}
	m, errM := atoiStrict(parts[1])
	if errM != nil || m < 0 || m > 59 {
		return TimeConversionResult{}, fmt.Errorf("invalid minute: %s", parts[1])
	}

	// Use the injectable nowFunc for the date context
	now := t.nowFunc()
	srcTime := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, srcLoc)
	dstTime := srcTime.In(dstLoc)

	_, srcOff := srcTime.Zone()
	_, dstOff := dstTime.Zone()
	diff := float64(dstOff-srcOff) / 3600
	// Format diffStr carefully to avoid excessive precision or trailing zeros
	var diffStr string
	if diff == float64(int(diff)) { // Check if it's a whole number
		diffStr = fmt.Sprintf("%+.0fh", diff)
	} else {
		diffStr = fmt.Sprintf("%+.2fh", diff)
		diffStr = strings.TrimRight(diffStr, "0") // Trim trailing zeros after decimal
		diffStr = strings.TrimRight(diffStr, ".") // Trim trailing decimal if it became "X."
	}

	return TimeConversionResult{
		Source: TimeResult{
			Timezone: srcTZ,
			Datetime: srcTime.Format(time.RFC3339),
			IsDST:    srcTime.IsDST(),
		},
		Target: TimeResult{
			Timezone: dstTZ,
			Datetime: dstTime.Format(time.RFC3339),
			IsDST:    dstTime.IsDST(),
		},
		TimeDifference: diffStr,
	}, nil
}

// ParseNatural uses the injectable nowFunc as the reference for 'when.Parser'
func (t *TimeServer) ParseNatural(expr, tz string) (TimeResult, error) {
	if tz == "" {
		tz = t.localTZ
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return TimeResult{}, fmt.Errorf("unknown time zone %s: %w", tz, err)
	}
	// Use the injectable nowFunc as the reference time for parsing
	nowForParsing := t.nowFunc().In(loc)
	res, err := t.parser.Parse(expr, nowForParsing)
	if err != nil || res == nil {
		// If err is not nil, include it. Otherwise, just state the expression couldn't be parsed.
		detailedError := fmt.Errorf("could not parse expression: %s", expr)
		if err != nil {
			detailedError = fmt.Errorf("could not parse expression '%s': %w", expr, err)
		}
		return TimeResult{}, detailedError
	}
	// The result from 'when.Parse' is relative to 'nowForParsing'.
	// We want the final time to be in the specified 'loc' (which is tz).
	out := res.Time.In(loc)
	return TimeResult{Timezone: tz, Datetime: out.Format(time.RFC3339), IsDST: out.IsDST()}, nil
}

/* ----- main ----- */
// ... (main function remains unchanged)
func main() {
	var transport, localTZ string
	var port int
	var showVer bool
	flag.StringVar(&transport, "transport", "stdio", "")
	flag.StringVar(&transport, "t", "stdio", "")
	flag.StringVar(&localTZ, "local-timezone", "", "")
	flag.StringVar(&localTZ, "l", "", "")
	flag.IntVar(&port, "port", 8080, "")
	flag.IntVar(&port, "p", 8080, "")
	flag.BoolVar(&showVer, "version", false, "print version and exit")
	flag.BoolVar(&showVer, "v", false, "print version and exit (shorthand)")
	flag.Parse()
	if showVer {
		fmt.Printf("%s %s\n", appName, version)
		return
	}

	ts := NewTimeServer(localTZ)

	s := server.NewMCPServer(
		appName, version,
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	getCurrent := mcp.NewTool(
		"get_current_time",
		mcp.WithDescription("Get the current time in a specific timezone."),
		mcp.WithString("timezone", mcp.Description("IANA timezone (optional).")),
	)

	convert := mcp.NewTool(
		"convert_time",
		mcp.WithDescription("Convert a HH:MM time between timezones."),
		mcp.WithString("source_timezone", mcp.Required()),
		mcp.WithString("time", mcp.Required()),
		mcp.WithString("target_timezone", mcp.Required()),
	)

	parseNL := mcp.NewTool(
		"parse_natural_time",
		mcp.WithDescription("Parse natural-language expressions (e.g., 'next Friday at noon')."),
		mcp.WithString("expression", mcp.Required()),
		mcp.WithString("timezone"),
	)

	s.AddTool(getCurrent, func(_ context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tz := r.GetString("timezone", "")
		res, err := ts.GetCurrentTime(tz)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(convert, func(_ context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		src, err := r.RequireString("source_timezone")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		hhmm, err := r.RequireString("time")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		dst, err := r.RequireString("target_timezone")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := ts.ConvertTime(src, hhmm, dst)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(parseNL, func(_ context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		expr, err := r.RequireString("expression")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tz := r.GetString("timezone", "")
		res, err := ts.ParseNatural(expr, tz)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})

	switch transport {
	case "stdio":
		log.Fatal(server.ServeStdio(s))
	case "sse":
		httpSrv := server.NewSSEServer(s, server.WithBaseURL(fmt.Sprintf("http://localhost:%d", port)))
		log.Fatal(httpSrv.Start(fmt.Sprintf(":%d", port)))
	default:
		log.Fatalf("unknown transport %q", transport)
	}
}
