package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Input validation constants
const (
	MaxGameNameLength = 50
	MaxTagLineLength  = 20
	MaxRegionLength   = 10
	MaxMatchIDLength  = 50
	MaxPUUIDLength    = 100
)

// Validation error types
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// Regular expressions for validation
var (
	// Game name: alphanumeric, spaces, hyphens, underscores, and some special characters
	gameNameRegex = regexp.MustCompile(`^[a-zA-Z0-9 _\-\.]+$`)

	// Tag line: alphanumeric and some special characters (no spaces)
	tagLineRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

	// Region: only lowercase letters and numbers
	regionRegex = regexp.MustCompile(`^[a-z0-9]+$`)

	// Match ID: alphanumeric with underscores and hyphens
	matchIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

	// PUUID: alphanumeric with hyphens (UUID format)
	puuidRegex = regexp.MustCompile(`^[a-zA-Z0-9\-]+$`)
)

// ValidateGameName validates a League of Legends game name
func ValidateGameName(gameName string) error {
	if gameName == "" {
		return ValidationError{Field: "gameName", Message: "game name cannot be empty"}
	}

	if len(gameName) > MaxGameNameLength {
		return ValidationError{Field: "gameName", Message: fmt.Sprintf("game name cannot exceed %d characters", MaxGameNameLength)}
	}

	// Check for valid characters
	if !gameNameRegex.MatchString(gameName) {
		return ValidationError{Field: "gameName", Message: "game name contains invalid characters"}
	}

	// Check for excessive whitespace
	if strings.TrimSpace(gameName) != gameName {
		return ValidationError{Field: "gameName", Message: "game name cannot start or end with whitespace"}
	}

	// Check for consecutive spaces
	if strings.Contains(gameName, "  ") {
		return ValidationError{Field: "gameName", Message: "game name cannot contain consecutive spaces"}
	}

	return nil
}

// ValidateTagLine validates a League of Legends tag line
func ValidateTagLine(tagLine string) error {
	if tagLine == "" {
		return ValidationError{Field: "tagLine", Message: "tag line cannot be empty"}
	}

	if len(tagLine) > MaxTagLineLength {
		return ValidationError{Field: "tagLine", Message: fmt.Sprintf("tag line cannot exceed %d characters", MaxTagLineLength)}
	}

	// Check for valid characters
	if !tagLineRegex.MatchString(tagLine) {
		return ValidationError{Field: "tagLine", Message: "tag line contains invalid characters"}
	}

	return nil
}

// ValidateRegion validates a League of Legends region code
func ValidateRegion(region string) error {
	if region == "" {
		return ValidationError{Field: "region", Message: "region cannot be empty"}
	}

	if len(region) > MaxRegionLength {
		return ValidationError{Field: "region", Message: fmt.Sprintf("region cannot exceed %d characters", MaxRegionLength)}
	}

	// Convert to lowercase for consistency
	region = strings.ToLower(region)

	// Check for valid characters
	if !regionRegex.MatchString(region) {
		return ValidationError{Field: "region", Message: "region contains invalid characters"}
	}

	// Validate against known regions
	validRegions := map[string]bool{
		"na1":  true, // North America
		"euw1": true, // Europe West
		"eun1": true, // Europe Nordic & East
		"kr":   true, // Korea
		"br1":  true, // Brazil
		"la1":  true, // Latin America North
		"la2":  true, // Latin America South
		"oc1":  true, // Oceania
		"tr1":  true, // Turkey
		"ru":   true, // Russia
		"jp1":  true, // Japan
		"ph2":  true, // Philippines
		"sg2":  true, // Singapore
		"th2":  true, // Thailand
		"tw2":  true, // Taiwan
		"vn2":  true, // Vietnam
		"pbe1": true, // Public Beta Environment
	}

	if !validRegions[region] {
		return ValidationError{Field: "region", Message: "invalid region code"}
	}

	return nil
}

// ValidateMatchID validates a League of Legends match ID
func ValidateMatchID(matchID string) error {
	if matchID == "" {
		return ValidationError{Field: "matchID", Message: "match ID cannot be empty"}
	}

	if len(matchID) > MaxMatchIDLength {
		return ValidationError{Field: "matchID", Message: fmt.Sprintf("match ID cannot exceed %d characters", MaxMatchIDLength)}
	}

	// Check for valid characters
	if !matchIDRegex.MatchString(matchID) {
		return ValidationError{Field: "matchID", Message: "match ID contains invalid characters"}
	}

	return nil
}

// ValidatePUUID validates a Player Universally Unique Identifier
func ValidatePUUID(puuid string) error {
	if puuid == "" {
		return ValidationError{Field: "puuid", Message: "PUUID cannot be empty"}
	}

	return nil
}

// ValidateCount validates count parameters for API requests
func ValidateCount(countStr string, defaultValue, maxValue int) (int, error) {
	if countStr == "" {
		return defaultValue, nil
	}

	count, err := strconv.Atoi(countStr)
	if err != nil {
		return 0, ValidationError{Field: "count", Message: "count must be a valid integer"}
	}

	if count <= 0 {
		return 0, ValidationError{Field: "count", Message: "count must be greater than 0"}
	}

	if count > maxValue {
		return 0, ValidationError{Field: "count", Message: fmt.Sprintf("count cannot exceed %d", maxValue)}
	}

	return count, nil
}

// ValidateQueueID validates queue ID parameters
func ValidateQueueID(queueIDStr string, defaultValue int) (int, error) {
	if queueIDStr == "" {
		return defaultValue, nil
	}

	queueID, err := strconv.Atoi(queueIDStr)
	if err != nil {
		return 0, ValidationError{Field: "queueID", Message: "queue ID must be a valid integer"}
	}

	// Allow any non-negative integer for queue ID
	if queueID < 0 {
		return 0, ValidationError{Field: "queueID", Message: "queue ID must be non-negative"}
	}

	return queueID, nil
}

// SanitizeString removes potentially dangerous characters and normalizes the string
func SanitizeString(input string) string {
	// Remove null bytes and other control characters
	var result strings.Builder
	for _, r := range input {
		if r == 0 || (r < 32 && r != 9 && r != 10 && r != 13) {
			continue // Skip null bytes and most control characters
		}
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}

	// Trim whitespace
	return strings.TrimSpace(result.String())
}

// ValidateAndSanitizeInput performs comprehensive validation and sanitization
func ValidateAndSanitizeInput(gameName, tagLine, region string) (string, string, string, error) {
	// Sanitize inputs first
	gameName = SanitizeString(gameName)
	tagLine = SanitizeString(tagLine)
	region = SanitizeString(region)

	// Validate each field
	if err := ValidateGameName(gameName); err != nil {
		return "", "", "", err
	}

	if err := ValidateTagLine(tagLine); err != nil {
		return "", "", "", err
	}

	if err := ValidateRegion(region); err != nil {
		return "", "", "", err
	}

	// Normalize region to lowercase
	region = strings.ToLower(region)

	return gameName, tagLine, region, nil
}

// ValidateMatchInput validates match-related input parameters
func ValidateMatchInput(region, matchID string) (string, string, error) {
	// Sanitize inputs
	region = SanitizeString(region)
	matchID = SanitizeString(matchID)

	// Validate each field
	if err := ValidateRegion(region); err != nil {
		return "", "", err
	}

	if err := ValidateMatchID(matchID); err != nil {
		return "", "", err
	}

	// Normalize region to lowercase
	region = strings.ToLower(region)

	return region, matchID, nil
}

// IsValidBSONObjectID checks if a string could be a valid BSON ObjectID
func IsValidBSONObjectID(id string) bool {
	if len(id) != 24 {
		return false
	}

	for _, r := range id {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}

	return true
}

// PreventNoSQLInjection validates that input doesn't contain MongoDB operators
func PreventNoSQLInjection(input string) error {
	// Check for common MongoDB operators that could be used in injection attacks
	dangerousPatterns := []string{
		"$where", "$regex", "$ne", "$gt", "$lt", "$gte", "$lte",
		"$in", "$nin", "$exists", "$type", "$mod", "$all",
		"$elemMatch", "$size", "$or", "$and", "$not", "$nor",
		"javascript:", "function(", "eval(", "setTimeout(", "setInterval(",
	}

	lowerInput := strings.ToLower(input)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerInput, pattern) {
			return ValidationError{Field: "input", Message: "input contains potentially dangerous patterns"}
		}
	}

	return nil
}
