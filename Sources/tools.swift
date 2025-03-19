import AppKit
import Foundation
import JSONSchemaBuilder
@preconcurrency import MCPServer

// MARK: - Get Current Time Tool

/// Input for getting current time in a specific timezone.
@Schemable
struct GetCurrentTimeInput {
    @SchemaOptions(
        title: "Timezone",
        description:
            "IANA timezone name (e.g., 'America/New_York', 'Europe/London'). If empty or not provided, the system timezone will be used."
    )
    let timezone: String
}

// MARK: - Convert Time Tool

/// Input for converting time between timezones.
@Schemable
struct ConvertTimeInput {
    @SchemaOptions(
        title: "Source Timezone",
        description:
            "Source IANA timezone name (e.g., 'America/New_York', 'Europe/London'). If empty or not provided, the system timezone will be used."
    )
    let source_timezone: String

    @SchemaOptions(
        title: "Time",
        description: "Time to convert in 24-hour format (HH:MM)"
    )
    let time: String

    @SchemaOptions(
        title: "Target Timezone",
        description:
            "Target IANA timezone name (e.g., 'Asia/Tokyo', 'America/San_Francisco'). If empty or not provided, the system timezone will be used."
    )
    let target_timezone: String
}

// MARK: - Time Results

/// Model for time information
struct TimeResult: Codable {
    let timezone: String
    let datetime: String
    let is_dst: Bool
}

/// Model for time conversion result
struct TimeConversionResult: Codable {
    let source: TimeResult
    let target: TimeResult
    let time_difference: String
}

// MARK: - Time Utility Functions

// Store the system timezone
private let localTimezone: TimeZone = TimeZone.current

/// Get timezone information
func getTimeZone(_ timezoneName: String) throws -> TimeZone {
    if let timezone = TimeZone(identifier: timezoneName) {
        return timezone
    }
    throw NSError(
        domain: "TimeZoneError", code: 1, userInfo: [NSLocalizedDescriptionKey: "Invalid timezone: \(timezoneName)"])
}

/// Get local timezone
func getLocalTimeZone() -> TimeZone {
    return TimeZone.current
}

// MARK: - Time Tools

let get_current_time = Tool(
    name: "get_current_time",
    description: "Get current time in a specific timezones"
) { (input: GetCurrentTimeInput) async throws -> [TextContentOrImageContentOrEmbeddedResource] in
    // If the input timezone is empty, use the system timezone
    let timezoneId = input.timezone.isEmpty ? TimeZone.current.identifier : input.timezone
    let timezone = try getTimeZone(timezoneId)

    // Get current time in the specified timezone
    let date = Date()
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    formatter.timeZone = timezone

    // Create the result
    let result = TimeResult(
        timezone: timezoneId,
        datetime: formatter.string(from: date),
        is_dst: timezone.isDaylightSavingTime(for: date)
    )

    // Convert to JSON
    let encoder = JSONEncoder()
    encoder.outputFormatting = [.prettyPrinted]
    let jsonData = try encoder.encode(result)
    let jsonString = String(data: jsonData, encoding: .utf8)!

    return [.text(TextContent(text: jsonString))]
}

let convert_time = Tool(
    name: "convert_time",
    description: "Convert time between timezones"
) { (input: ConvertTimeInput) async throws -> [TextContentOrImageContentOrEmbeddedResource] in
    // If the source timezone is empty, use the system timezone
    let sourceTimezoneId = input.source_timezone.isEmpty ? TimeZone.current.identifier : input.source_timezone
    // If the target timezone is empty, use the system timezone
    let targetTimezoneId = input.target_timezone.isEmpty ? TimeZone.current.identifier : input.target_timezone

    // Get source and target timezones
    let sourceTimezone = try getTimeZone(sourceTimezoneId)
    let targetTimezone = try getTimeZone(targetTimezoneId)

    // Parse the input time (HH:MM)
    let timeComponents = input.time.split(separator: ":")
    guard timeComponents.count == 2,
        let hour = Int(timeComponents[0]),
        let minute = Int(timeComponents[1]),
        hour >= 0, hour < 24,
        minute >= 0, minute < 60
    else {
        throw NSError(
            domain: "TimeFormatError", code: 2,
            userInfo: [NSLocalizedDescriptionKey: "Invalid time format. Expected HH:MM [24-hour format]"])
    }

    // Get current date components in the source timezone
    let now = Date()
    var calendar = Calendar.current
    calendar.timeZone = sourceTimezone

    // Create date with the specified time in source timezone
    var components = calendar.dateComponents([.year, .month, .day], from: now)
    components.hour = hour
    components.minute = minute

    guard let sourceDate = calendar.date(from: components) else {
        throw NSError(
            domain: "DateError", code: 3, userInfo: [NSLocalizedDescriptionKey: "Failed to create date from components"]
        )
    }

    // Create formatted dates for both timezones
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]

    formatter.timeZone = sourceTimezone
    let sourceDateString = formatter.string(from: sourceDate)

    formatter.timeZone = targetTimezone
    let targetDateString = formatter.string(from: sourceDate)

    // Calculate time difference
    let sourceOffset = sourceTimezone.secondsFromGMT(for: sourceDate)
    let targetOffset = targetTimezone.secondsFromGMT(for: sourceDate)
    let hoursDifference = Double(targetOffset - sourceOffset) / 3600.0

    // Format the time difference string
    let timeDiffStr: String
    if hoursDifference == floor(hoursDifference) {
        timeDiffStr = String(format: "%+.1fh", hoursDifference)
    } else {
        let formatter = NumberFormatter()
        formatter.minimumFractionDigits = 1
        formatter.maximumFractionDigits = 2
        let formattedNumber = formatter.string(from: NSNumber(value: hoursDifference)) ?? String(hoursDifference)
        timeDiffStr = formattedNumber.hasSuffix(".0") ? "\(formattedNumber.dropLast(2))h" : "\(formattedNumber)h"
    }

    // Create the result
    let result = TimeConversionResult(
        source: TimeResult(
            timezone: sourceTimezoneId,
            datetime: sourceDateString,
            is_dst: sourceTimezone.isDaylightSavingTime(for: sourceDate)
        ),
        target: TimeResult(
            timezone: targetTimezoneId,
            datetime: targetDateString,
            is_dst: targetTimezone.isDaylightSavingTime(for: sourceDate)
        ),
        time_difference: timeDiffStr
    )

    // Convert to JSON
    let encoder = JSONEncoder()
    encoder.outputFormatting = [.prettyPrinted]
    let jsonData = try encoder.encode(result)
    let jsonString = String(data: jsonData, encoding: .utf8)!

    return [.text(TextContent(text: jsonString))]
}
