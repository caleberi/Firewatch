package main

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"
)

// ValidatorFunc defines the signature for validation functions
type ValidatorFunc func(value interface{}) error

// ValidationError holds field-specific validation errors
type ValidationError map[string][]error

func (v ValidationError) ToJSON() ([]byte, error) {
	if len(v) == 0 {
		return nil, nil
	}

	jsonErrors := make(map[string][]string)
	for field, errs := range v {
		for _, err := range errs {
			jsonErrors[field] = append(jsonErrors[field], err.Error())
		}
	}

	return json.Marshal(jsonErrors)
}

// Validator holds the struct to validate and accumulated errors
type Validator[T any] struct {
	target T
	errors ValidationError
}

// NewValidator creates a new Validator instance
func NewValidator[T any](target T) *Validator[T] {
	return &Validator[T]{
		target: target,
		errors: make(ValidationError),
	}
}

// ValidateField adds validation for a specific field
func (v *Validator[T]) ValidateField(fieldName string, validators ...ValidatorFunc) *Validator[T] {
	val := reflect.ValueOf(v.target)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		v.errors[fieldName] = append(v.errors[fieldName], fmt.Errorf("target is not a struct"))
		return v
	}

	field := val.FieldByName(fieldName)
	if !field.IsValid() {
		v.errors[fieldName] = append(v.errors[fieldName], fmt.Errorf("field %s not found", fieldName))
		return v
	}

	for _, validator := range validators {
		if err := validator(field.Interface()); err != nil {
			v.errors[fieldName] = append(v.errors[fieldName], err)
		}
	}

	return v
}

// Errors returns the validation errors, or nil if no errors
func (v *Validator[T]) Errors() ValidationError {
	if len(v.errors) == 0 {
		return nil
	}
	return v.errors
}

// ToJSON converts validation errors to JSON
func (v *Validator[T]) ToJSON() ([]byte, error) {
	if len(v.errors) == 0 {
		return nil, nil
	}

	jsonErrors := make(map[string][]string)
	for field, errs := range v.errors {
		for _, err := range errs {
			jsonErrors[field] = append(jsonErrors[field], err.Error())
		}
	}

	return json.Marshal(jsonErrors)
}

func IsEmpty(value any) error {
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("field cannot be empty")
		}
	case *string:
		if v == nil || strings.TrimSpace(*v) == "" {
			return fmt.Errorf("field cannot be empty or nil")
		}
	case []string:
		if len(v) == 0 {
			return fmt.Errorf("slice cannot be empty")
		}
	case *[]string:
		if v == nil || len(*v) == 0 {
			return fmt.Errorf("slice cannot be empty or nil")
		}
	default:
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			if rv.IsNil() {
				return fmt.Errorf("slice cannot be nil")
			}
			if rv.Len() == 0 {
				return fmt.Errorf("slice or array cannot be empty")
			}
			return nil
		}
		return fmt.Errorf("unsupported type for IsEmpty")
	}
	return nil
}

func IsSupported(values ...string) ValidatorFunc {
	return func(value any) error {
		switch v := value.(type) {
		case string:
			if !slices.Contains(values, strings.TrimSpace(v)) {
				return fmt.Errorf("value provided to field not supported, only %v are supported", values)
			}
		case *string:
			if v == nil || !slices.Contains(values, strings.TrimSpace(*v)) {
				return fmt.Errorf("value provided to field not supported, only %v are supported", values)
			}
		}
		return nil
	}
}

// IsEmail validates if the value is a valid email address
func IsEmail(value any) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	switch v := value.(type) {
	case string:
		if !emailRegex.MatchString(strings.TrimSpace(v)) {
			return fmt.Errorf("invalid email format")
		}
	case *string:
		if v == nil || !emailRegex.MatchString(strings.TrimSpace(*v)) {
			return fmt.Errorf("invalid email format or nil")
		}
	default:
		return fmt.Errorf("unsupported type for IsEmail")
	}
	return nil
}

// IsURL validates if the value is a valid URL
func IsURL(value any) error {
	urlRegex := regexp.MustCompile(`^(https?|ftp)://[^\s/$.?#].[^\s]*$`)
	switch v := value.(type) {
	case string:
		if !urlRegex.MatchString(strings.TrimSpace(v)) {
			return fmt.Errorf("invalid URL format")
		}
	case *string:
		if v == nil || !urlRegex.MatchString(strings.TrimSpace(*v)) {
			return fmt.Errorf("invalid URL format or nil")
		}
	default:
		return fmt.Errorf("unsupported type for IsURL")
	}
	return nil
}

// IsIPv4 validates if the value is a valid IPv4 address
func IsIPv4(value interface{}) error {
	switch v := value.(type) {
	case string:
		if ip := net.ParseIP(strings.TrimSpace(v)); ip == nil || ip.To4() == nil {
			return fmt.Errorf("invalid IPv4 address")
		}
	case *string:
		if v == nil || net.ParseIP(strings.TrimSpace(*v)) == nil || net.ParseIP(strings.TrimSpace(*v)).To4() == nil {
			return fmt.Errorf("invalid IPv4 address or nil")
		}
	default:
		return fmt.Errorf("unsupported type for IsIPv4")
	}
	return nil
}

// MinLength validates if the value meets a minimum length requirement
func MinLength(min int) ValidatorFunc {
	return func(value interface{}) error {
		switch v := value.(type) {
		case string:
			if len(strings.TrimSpace(v)) < min {
				return fmt.Errorf("length must be at least %d characters", min)
			}
		case *string:
			if v == nil || len(strings.TrimSpace(*v)) < min {
				return fmt.Errorf("length must be at least %d characters or nil", v)
			}
		case []string:
			if len(v) < min {
				return fmt.Errorf("slice must have at least %d elements", min)
			}
		case *[]string:
			if v == nil || len(*v) < min {
				return fmt.Errorf("slice must have at least %d elements or nil", len(*v))
			}
		default:
			rv := reflect.ValueOf(value)
			if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
				if rv.IsNil() {
					return fmt.Errorf("slice cannot be nil")
				}
				if rv.Len() < min {
					return fmt.Errorf("slice or array must have at least %d elements", min)
				}
				return nil
			}
			return fmt.Errorf("unsupported type for MinLength")
		}
		return nil
	}
}

// IsDate validates if the value is a valid date in the format YYYY-MM-DD
func IsDate(value interface{}) error {
	const layout = "2006-01-02"
	switch v := value.(type) {
	case string:
		_, err := time.Parse(layout, strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("invalid date format, expected YYYY-MM-DD")
		}
	case *string:
		if v == nil {
			return fmt.Errorf("date cannot be nil")
		}
		_, err := time.Parse(layout, strings.TrimSpace(*v))
		if err != nil {
			return fmt.Errorf("invalid date format, expected YYYY-MM-DD")
		}
	default:
		return fmt.Errorf("unsupported type for IsDate")
	}
	return nil
}

// ContainsAny validates if the value contains any of the specified characters
func ContainsAny(chars string) ValidatorFunc {
	return func(value interface{}) error {
		switch v := value.(type) {
		case string:
			if !strings.ContainsAny(strings.TrimSpace(v), chars) {
				return fmt.Errorf("must contain at least one of the characters: %s", chars)
			}
		case *string:
			if v == nil || !strings.ContainsAny(strings.TrimSpace(*v), chars) {
				return fmt.Errorf("must contain at least one of the characters: %s or nil", chars)
			}
		default:
			return fmt.Errorf("unsupported type for ContainsAny")
		}
		return nil
	}
}

// IsAlpha validates if the value contains only alphabetic characters
func IsAlpha(value interface{}) error {
	alphaRegex := regexp.MustCompile(`^[a-zA-Z]+$`)
	switch v := value.(type) {
	case string:
		if !alphaRegex.MatchString(strings.TrimSpace(v)) {
			return fmt.Errorf("must contain only alphabetic characters")
		}
	case *string:
		if v == nil || !alphaRegex.MatchString(strings.TrimSpace(*v)) {
			return fmt.Errorf("must contain only alphabetic characters or nil")
		}
	default:
		return fmt.Errorf("unsupported type for IsAlpha")
	}
	return nil
}

// IsUUID validates if the value is a valid UUID (any version)
func IsUUID(value interface{}) error {
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	switch v := value.(type) {
	case string:
		if !uuidRegex.MatchString(strings.TrimSpace(v)) {
			return fmt.Errorf("invalid UUID format")
		}
	case *string:
		if v == nil || !uuidRegex.MatchString(strings.TrimSpace(*v)) {
			return fmt.Errorf("invalid UUID format or nil")
		}
	default:
		return fmt.Errorf("unsupported type for IsUUID")
	}
	return nil
}

// MaxLength validates if the value does not exceed a maximum length
func MaxLength(max int) ValidatorFunc {
	return func(value interface{}) error {
		switch v := value.(type) {
		case string:
			if len(strings.TrimSpace(v)) > max {
				return fmt.Errorf("length must not exceed %d characters", max)
			}
		case *string:
			if v == nil {
				return nil // Nil is considered valid as it has no length
			}
			if len(strings.TrimSpace(*v)) > max {
				return fmt.Errorf("length must not exceed %d characters", max)
			}
		case []string:
			if len(v) > max {
				return fmt.Errorf("slice must not have more than %d elements", max)
			}
		case *[]string:
			if v == nil {
				return nil // Nil is considered valid
			}
			if len(*v) > max {
				return fmt.Errorf("slice must not have more than %d elements", max)
			}
		default:
			rv := reflect.ValueOf(value)
			if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
				if rv.IsNil() {
					return nil // Nil is considered valid
				}
				if rv.Len() > max {
					return fmt.Errorf("slice or array must not have more than %d elements", max)
				}
				return nil
			}
			return fmt.Errorf("unsupported type for MaxLength")
		}
		return nil
	}
}

// IsNumeric validates if the value contains only numeric characters
func IsNumeric(value interface{}) error {
	numericRegex := regexp.MustCompile(`^[0-9]+$`)
	switch v := value.(type) {
	case string:
		if !numericRegex.MatchString(strings.TrimSpace(v)) {
			return fmt.Errorf("must contain only numeric characters")
		}
	case *string:
		if v == nil || !numericRegex.MatchString(strings.TrimSpace(*v)) {
			return fmt.Errorf("must contain only numeric characters or nil")
		}
	default:
		return fmt.Errorf("unsupported type for IsNumeric")
	}
	return nil
}

// IsInRange validates if a numeric value is within a specified range
func IsInRange(min, max float64) ValidatorFunc {
	return func(value interface{}) error {
		var val float64
		switch v := value.(type) {
		case int:
			val = float64(v)
		case int8:
			val = float64(v)
		case int16:
			val = float64(v)
		case int32:
			val = float64(v)
		case int64:
			val = float64(v)
		case uint:
			val = float64(v)
		case uint8:
			val = float64(v)
		case uint16:
			val = float64(v)
		case uint32:
			val = float64(v)
		case uint64:
			val = float64(v)
		case float32:
			val = float64(v)
		case float64:
			val = v
		case *float64:
			if v == nil {
				return fmt.Errorf("value cannot be nil")
			}
			val = *v
		default:
			return fmt.Errorf("unsupported type for IsInRange")
		}
		if val < min || val > max {
			return fmt.Errorf("value must be between %v and %v", min, max)
		}
		return nil
	}
}

// IsMAC validates if the value is a valid MAC address
func IsMAC(value interface{}) error {
	macRegex := regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)
	switch v := value.(type) {
	case string:
		if !macRegex.MatchString(strings.TrimSpace(v)) {
			return fmt.Errorf("invalid MAC address format")
		}
	case *string:
		if v == nil || !macRegex.MatchString(strings.TrimSpace(*v)) {
			return fmt.Errorf("invalid MAC address format or nil")
		}
	default:
		return fmt.Errorf("unsupported type for IsMAC")
	}
	return nil
}

// IsAfter validates if a date is after a specified date
func IsAfter(compareDate time.Time) ValidatorFunc {
	return func(value interface{}) error {
		const layout = "2006-01-02"
		var t time.Time
		var err error
		switch v := value.(type) {
		case string:
			t, err = time.Parse(layout, strings.TrimSpace(v))
			if err != nil {
				return fmt.Errorf("invalid date format, expected YYYY-MM-DD")
			}
		case *string:
			if v == nil {
				return fmt.Errorf("date cannot be nil")
			}
			t, err = time.Parse(layout, strings.TrimSpace(*v))
			if err != nil {
				return fmt.Errorf("invalid date format, expected YYYY-MM-DD")
			}
		case time.Time:
			t = v
		case *time.Time:
			if v == nil {
				return fmt.Errorf("date cannot be nil")
			}
			t = *v
		default:
			return fmt.Errorf("unsupported type for IsAfter")
		}
		if !t.After(compareDate) {
			return fmt.Errorf("date must be after %s", compareDate.Format(layout))
		}
		return nil
	}
}

// IsAlphanumeric validates if the value contains only alphanumeric characters
func IsAlphanumeric(value interface{}) error {
	alphanumericRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	switch v := value.(type) {
	case string:
		if !alphanumericRegex.MatchString(strings.TrimSpace(v)) {
			return fmt.Errorf("must contain only alphanumeric characters")
		}
	case *string:
		if v == nil || !alphanumericRegex.MatchString(strings.TrimSpace(*v)) {
			return fmt.Errorf("must contain only alphanumeric characters or nil")
		}
	default:
		return fmt.Errorf("unsupported type for IsAlphanumeric")
	}
	return nil
}

// IsJSON validates if the value is valid JSON
func IsJSON(value interface{}) error {
	switch v := value.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return fmt.Errorf("JSON string cannot be empty")
		}
		var js interface{}
		if err := json.Unmarshal([]byte(trimmed), &js); err != nil {
			return fmt.Errorf("invalid JSON format: %v", err)
		}
	case *string:
		if v == nil {
			return fmt.Errorf("JSON string cannot be nil")
		}
		trimmed := strings.TrimSpace(*v)
		if trimmed == "" {
			return fmt.Errorf("JSON string cannot be empty")
		}
		var js interface{}
		if err := json.Unmarshal([]byte(trimmed), &js); err != nil {
			return fmt.Errorf("invalid JSON format: %v", err)
		}
	default:
		return fmt.Errorf("unsupported type for IsJSON")
	}
	return nil
}

// IsHexColor validates if the value is a valid hexadecimal color code
func IsHexColor(value interface{}) error {
	hexColorRegex := regexp.MustCompile(`^#([0-9a-fA-F]{6}|[0-9a-fA-F]{3})$`)
	switch v := value.(type) {
	case string:
		if !hexColorRegex.MatchString(strings.TrimSpace(v)) {
			return fmt.Errorf("invalid hex color format, expected #RRGGBB or #RGB")
		}
	case *string:
		if v == nil || !hexColorRegex.MatchString(strings.TrimSpace(*v)) {
			return fmt.Errorf("invalid hex color format or nil, expected #RRGGBB or #RGB")
		}
	default:
		return fmt.Errorf("unsupported type for IsHexColor")
	}
	return nil
}

// IsPositive validates if a numeric value is positive
func IsPositive(value interface{}) error {
	var val float64
	switch v := value.(type) {
	case int:
		val = float64(v)
	case int8:
		val = float64(v)
	case int16:
		val = float64(v)
	case int32:
		val = float64(v)
	case int64:
		val = float64(v)
	case uint:
		val = float64(v)
	case uint8:
		val = float64(v)
	case uint16:
		val = float64(v)
	case uint32:
		val = float64(v)
	case uint64:
		val = float64(v)
	case float32:
		val = float64(v)
	case float64:
		val = v
	case *float64:
		if v == nil {
			return fmt.Errorf("value cannot be nil")
		}
		val = *v
	default:
		return fmt.Errorf("unsupported type for IsPositive")
	}
	if val <= 0 {
		return fmt.Errorf("value must be positive")
	}
	return nil
}

// StartsWith validates if a string starts with a specified prefix
func StartsWith(prefix string) ValidatorFunc {
	return func(value interface{}) error {
		switch v := value.(type) {
		case string:
			if !strings.HasPrefix(strings.TrimSpace(v), prefix) {
				return fmt.Errorf("must start with %q", prefix)
			}
		case *string:
			if v == nil || !strings.HasPrefix(strings.TrimSpace(*v), prefix) {
				return fmt.Errorf("must start with %q or nil", prefix)
			}
		default:
			return fmt.Errorf("unsupported type for StartsWith")
		}
		return nil
	}
}

// EndsWith validates if a string ends with a specified suffix
func EndsWith(suffix string) ValidatorFunc {
	return func(value interface{}) error {
		switch v := value.(type) {
		case string:
			if !strings.HasSuffix(strings.TrimSpace(v), suffix) {
				return fmt.Errorf("must end with %q", suffix)
			}
		case *string:
			if v == nil || !strings.HasSuffix(strings.TrimSpace(*v), suffix) {
				return fmt.Errorf("must end with %q or nil", suffix)
			}
		default:
			return fmt.Errorf("unsupported type for EndsWith")
		}
		return nil
	}
}

// IsUnique validates if all elements in a slice are unique
func IsUnique(value interface{}) error {
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return fmt.Errorf("slice cannot be nil")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return fmt.Errorf("unsupported type for IsUnique, expected slice or array")
	}

	seen := make(map[interface{}]struct{})
	for i := 0; i < rv.Len(); i++ {
		val := rv.Index(i).Interface()
		if _, exists := seen[val]; exists {
			return fmt.Errorf("slice contains duplicate value: %v", val)
		}
		seen[val] = struct{}{}
	}
	return nil
}
