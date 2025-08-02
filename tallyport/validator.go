package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
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
