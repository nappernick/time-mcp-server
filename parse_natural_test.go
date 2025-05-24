// parse_natural_test.go
package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/en"
)

// TestWhenParserWithFixedNow tests the underlying 'when.Parser' with a fixed reference time.
// This remains largely the same, as it tests the 'when' library directly.
func TestWhenParserWithFixedNow(t *testing.T) {
	fixedNow := time.Date(2025, 5, 17, 8, 0, 0, 0, time.FixedZone("FixedUTC-5", -5*3600))
	parser := when.New(nil)
	parser.Add(en.All...)

	cases := []struct {
		name      string
		expr      string
		wantYear  int
		wantMonth time.Month
		wantDay   int
		wantHour  int
		wantMin   int
	}{
		{"nextFridayNoon", "next Friday at noon", 2025, time.May, 23, 12, 0},
		{"tomorrow8pm", "tomorrow at 8pm", 2025, time.May, 18, 20, 0},
		{"threeDaysFromNow", "3 days from now", 2025, time.May, 20, 8, 0},
		{"lastMonday9am", "last Monday at 9am", 2025, time.May, 12, 9, 0},
		{"aug15_2024", "August 15th, 2024 10:30", 2024, time.August, 15, 10, 30},
		{"specificTimeToday", "today at 2:15 PM", 2025, time.May, 17, 14, 15},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := parser.Parse(tc.expr, fixedNow)
			if err != nil {
				t.Fatalf("parser.Parse(%q) error: %v", tc.expr, err)
			}
			if res == nil {
				t.Fatalf("parser.Parse(%q) returned nil", tc.expr)
			}
			got := res.Time
			if got.Year() != tc.wantYear ||
				got.Month() != tc.wantMonth ||
				got.Day() != tc.wantDay ||
				got.Hour() != tc.wantHour ||
				got.Minute() != tc.wantMin {
				t.Errorf("expr %q (fixedNow %v):\n  got: %v\n want: %d-%02d-%02d %02d:%02d",
					tc.expr, fixedNow.Format(time.RFC3339), got.Format(time.RFC3339),
					tc.wantYear, tc.wantMonth, tc.wantDay, tc.wantHour, tc.wantMin)
			}
		})
	}
}

// TestTimeServerParseNatural_Deterministic tests the full TimeServer.ParseNatural method
// using an injected, fixed "now" time for deterministic results.
func TestTimeServerParseNatural_Deterministic(t *testing.T) {
	// Define a fixed "now" for all tests in this function.
	// Saturday, May 17, 2025, 10:30:00 AM in America/New_York (EDT, UTC-4)
	locNY, _ := time.LoadLocation("America/New_York")
	fixedNow := time.Date(2025, 5, 17, 10, 30, 0, 0, locNY)

	// Create a TimeServer instance and inject our fixed "now"
	ts := NewTimeServer("UTC") // Default server TZ doesn't matter much if tests specify TZ
	ts.forTesting_SetNowFunc(func() time.Time {
		return fixedNow
	})

	// Helper to check TimeResult
	checkResult := func(t *testing.T, res TimeResult, expr, expectedOutputTZ string, checkTime func(parsedTime time.Time, loc *time.Location)) {
		t.Helper()
		if res.Timezone != expectedOutputTZ {
			t.Errorf("expr %q: expected output timezone %q, got %q", expr, expectedOutputTZ, res.Timezone)
		}
		parsedTimeUTC, err := time.Parse(time.RFC3339, res.Datetime) // res.Datetime is always UTC (RFC3339)
		if err != nil {
			t.Fatalf("expr %q: could not parse returned datetime %q: %v", expr, res.Datetime, err)
		}

		// Load the location for the expectedOutputTZ to perform checks in that zone
		outputLoc, err := time.LoadLocation(expectedOutputTZ)
		if err != nil {
			t.Fatalf("expr %q: could not load location for expectedOutputTZ %q: %v", expr, expectedOutputTZ, err)
		}

		if checkTime != nil {
			checkTime(parsedTimeUTC, outputLoc)
		}
	}

	t.Run("specificDateTimeWithExplicitTZ", func(t *testing.T) {
		expr := "July 4, 2026 10:00 AM"
		parseAsTZ := "America/Los_Angeles" // Parse expression as if it's LA time
		res, err := ts.ParseNatural(expr, parseAsTZ)
		if err != nil {
			t.Fatalf("ParseNatural(%q, %q) error: %v", expr, parseAsTZ, err)
		}
		checkResult(t, res, expr, parseAsTZ, func(ptUTC time.Time, locLA *time.Location) {
			laTime := ptUTC.In(locLA)
			if laTime.Year() != 2026 || laTime.Month() != time.July || laTime.Day() != 4 || laTime.Hour() != 10 {
				t.Errorf("Expected 2026-07-04 10:00 in %s, got %v", parseAsTZ, laTime.Format(time.RFC3339))
			}
			// July 4th in LA is usually DST
			expectedDST := time.Date(2026, time.July, 4, 10, 0, 0, 0, locLA).IsDST()
			if res.IsDST != expectedDST {
				t.Errorf("DST mismatch for %s in %s: result %v, expected %v", expr, parseAsTZ, res.IsDST, expectedDST)
			}
		})
	})

	t.Run("relativeTomorrowUsingFixedNowInUTC", func(t *testing.T) {
		expr := "tomorrow at 9:30am"
		parseAsTZ := "UTC" // Parse relative to fixedNow (converted to UTC)
		res, err := ts.ParseNatural(expr, parseAsTZ)
		if err != nil {
			t.Fatalf("ParseNatural(%q, %q) error: %v", expr, parseAsTZ, err)
		}
		checkResult(t, res, expr, parseAsTZ, func(ptUTC time.Time, locUTC *time.Location) {
			// fixedNow is May 17, 2025, 10:30:00 EDT (UTC-4) => 14:30:00 UTC
			// Tomorrow from fixedNow (UTC) is May 18, 2025
			expectedTimeUTC := time.Date(2025, time.May, 18, 9, 30, 0, 0, locUTC)
			if !ptUTC.Equal(expectedTimeUTC) {
				t.Errorf("Expected %v (tomorrow 9:30 UTC from fixedNow %v UTC), got %v",
					expectedTimeUTC.Format(time.RFC3339),
					fixedNow.In(locUTC).Format(time.RFC3339),
					ptUTC.Format(time.RFC3339))
			}
		})
	})

	t.Run("relativeNextMondayUsingFixedNowInChicago", func(t *testing.T) {
		expr := "next monday 2pm"
		parseAsTZ := "America/Chicago" // Parse relative to fixedNow (converted to Chicago time)
		res, err := ts.ParseNatural(expr, parseAsTZ)
		if err != nil {
			t.Fatalf("ParseNatural(%q, %q) error: %v", expr, parseAsTZ, err)
		}
		checkResult(t, res, expr, parseAsTZ, func(ptUTC time.Time, locChicago *time.Location) {
			// fixedNow is Sat, May 17, 2025, 10:30 EDT. In Chicago (CDT, UTC-5), this is 09:30 CDT.
			// "Next Monday" from Sat, May 17 is Mon, May 19.
			// Expected time is Mon, May 19, 2025, 2:00 PM (14:00) in Chicago.
			expectedTimeChicago := time.Date(2025, time.May, 19, 14, 0, 0, 0, locChicago)
			if !ptUTC.Equal(expectedTimeChicago.In(time.UTC)) {
				t.Errorf("Expected %v (next Monday 2pm Chicago from fixedNow %v Chicago), got %v (UTC)",
					expectedTimeChicago.Format(time.RFC3339),
					fixedNow.In(locChicago).Format(time.RFC3339),
					ptUTC.Format(time.RFC3339))
			}
			expectedDST := expectedTimeChicago.IsDST()
			if res.IsDST != expectedDST {
				t.Errorf("DST mismatch: result %v, expected %v for %v", res.IsDST, expectedDST, expectedTimeChicago)
			}
		})
	})

	t.Run("noTimezoneUsesServerDefaultWithFixedNow", func(t *testing.T) {
		// Recreate TimeServer with a specific default and inject fixedNow
		tsChicagoDefault := NewTimeServer("America/Chicago")
		tsChicagoDefault.forTesting_SetNowFunc(func() time.Time { return fixedNow })

		expr := "January 10, 2027 3:00 PM"
		res, err := tsChicagoDefault.ParseNatural(expr, "") // Empty tz string, should use server's default
		if err != nil {
			t.Fatalf("ParseNatural(%q, \"\") error: %v", expr, err)
		}
		expectedOutputTZ := "America/Chicago"
		checkResult(t, res, expr, expectedOutputTZ, func(ptUTC time.Time, locChicago *time.Location) {
			chicagoTime := ptUTC.In(locChicago)
			if chicagoTime.Year() != 2027 || chicagoTime.Month() != time.January || chicagoTime.Day() != 10 || chicagoTime.Hour() != 15 {
				t.Errorf("Expected 2027-01-10 15:00 in Chicago, got %v", chicagoTime.Format(time.RFC3339))
			}
		})
	})

	t.Run("invalidTimezoneError", func(t *testing.T) {
		expr := "now"
		tz := "Invalid/Timezone"
		_, err := ts.ParseNatural(expr, tz) // ts uses fixedNow
		if err == nil {
			t.Fatalf("Expected error for invalid timezone %q, got nil", tz)
		}
		if !strings.Contains(err.Error(), "unknown time zone Invalid/Timezone") {
			t.Errorf("Expected error to contain 'unknown time zone Invalid/Timezone', got: %v", err)
		}
	})

	t.Run("unparseableExpressionError", func(t *testing.T) {
		expr := "this is not a date at all"
		tz := "UTC"
		_, err := ts.ParseNatural(expr, tz) // ts uses fixedNow
		if err == nil {
			t.Fatalf("Expected error for unparseable expression %q, got nil", expr)
		}
		expectedErrorSubString := fmt.Sprintf("could not parse expression '%s'", expr)
		if !strings.Contains(err.Error(), expectedErrorSubString) {
			t.Errorf("Expected error to contain '%s', got: %v", expectedErrorSubString, err)
		}
	})

	t.Run("dstSpringForwardFixedNow", func(t *testing.T) {
		// Fixed "now" for this specific subtest, if needed, or use the main fixedNow.
		// The main fixedNow (May 17, 2025) is not near a DST transition.
		// Let's set a "now" that is just before a known DST transition for these specific expressions.
		locNYTest, _ := time.LoadLocation("America/New_York")
		// March 9, 2025, is when DST starts in NY. Let's set "now" to March 8, 2025.
		dstTestFixedNow := time.Date(2025, time.March, 8, 10, 0, 0, 0, locNYTest)
		tsDSTTest := NewTimeServer("America/New_York")
		tsDSTTest.forTesting_SetNowFunc(func() time.Time { return dstTestFixedNow })

		tzNY := "America/New_York"

		exprBefore := "March 9, 2025, 1:59 AM" // This is 1:59 AM EST
		resBefore, errB := tsDSTTest.ParseNatural(exprBefore, tzNY)
		if errB != nil {
			t.Fatalf("Error parsing %q: %v", exprBefore, errB)
		}
		checkResult(t, resBefore, exprBefore, tzNY, func(ptUTC time.Time, loc *time.Location) {
			nyTime := ptUTC.In(loc)
			if nyTime.Hour() != 1 || nyTime.Minute() != 59 {
				t.Errorf("Expected 01:59, got %s", nyTime.Format("15:04"))
			}
			if nyTime.IsDST() {
				t.Errorf("%q: expected IsDST=false, got true", exprBefore)
			}
		})

		exprAfter := "March 9, 2025, 3:01 AM" // This is 3:01 AM EDT
		resAfter, errA := tsDSTTest.ParseNatural(exprAfter, tzNY)
		if errA != nil {
			t.Fatalf("Error parsing %q: %v", exprAfter, errA)
		}
		checkResult(t, resAfter, exprAfter, tzNY, func(ptUTC time.Time, loc *time.Location) {
			nyTime := ptUTC.In(loc)
			if nyTime.Hour() != 3 || nyTime.Minute() != 1 {
				t.Errorf("Expected 03:01, got %s", nyTime.Format("15:04"))
			}
			if !nyTime.IsDST() {
				t.Errorf("%q: expected IsDST=true, got false", exprAfter)
			}
		})

		// Time during the "phantom" hour. 'when' might shift this.
		// For "March 9, 2025, 2:30 AM" in NY, it doesn't exist.
		// `when` might parse this as 2:30 standard time, which then becomes 3:30 daylight time.
		exprDuring := "March 9, 2025, 2:30 AM"
		resDuring, errD := tsDSTTest.ParseNatural(exprDuring, tzNY)
		if errD != nil {
			t.Logf("Parsing %q (during DST spring forward) resulted in error (potentially expected for some parsers): %v", exprDuring, errD)
			// Depending on 'when's strictness, an error might be valid.
			// If 'when' is lenient and shifts, the below checks would apply.
			// For now, let's assume 'when' might error or shift it. If it errors, this test path is fine.
		} else {
			checkResult(t, resDuring, exprDuring, tzNY, func(ptUTC time.Time, loc *time.Location) {
				nyTime := ptUTC.In(loc)
				// Expectation: 2:30 AM EST becomes 3:30 AM EDT
				if nyTime.Hour() != 3 || nyTime.Minute() != 30 {
					t.Errorf("Expected %q to resolve to 3:30 AM EDT due to DST, got %s", exprDuring, nyTime.Format("15:04 MST"))
				}
				if !nyTime.IsDST() {
					t.Errorf("%q: expected IsDST=true after DST jump, got false", exprDuring)
				}
			})
		}
	})
}
