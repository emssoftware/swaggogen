package main

/*
This interface reflects the kinds of validations provided by the validator
package (gopkg.in/go-playground/validator.v9).

This interface is kept distinct from the rest for the sake of code
organization and documentation.

Some of the material in the comments is from the validator package (above)
and other material is from the JSON schema validation RFC
(http://json-schema.org/latest/json-schema-validation.html). The material is
annotated to avoid confusion, and some modification have been made for the
same reason.

For compatibility with the Go JSON spec library (github.com/go-openapi/spec),
the return types they are what they are.

The site,
https://spacetelescope.github.io/understanding-json-schema/reference/string.html,
may have clues to creating more interesting validations (email, urls, etc.).
*/
type Validator interface {

	/*
		Validator Package Documentation:

			This validates that the value is not the data types default zero
			value. For numbers ensures value is not zero. For strings ensures
			value is not "". For slices, maps, pointers, interfaces, channels
			and functions ensures the value is not nil.

		JSON Schema Validation RFC:

			6.17. required
				The value of this keyword MUST be an array. Elements of this array, if any, MUST be strings, and MUST be unique.
				An object instance is valid against this keyword if every item in the array is the name of a property in the instance.
				Omitting this keyword has the same behavior as an empty array.

		Returns -1 when no length is enforced.
	*/
	IsRequired() bool

	/*
		Validator Package Documentation:

			For numbers, length will ensure that the value is equal to the
			parameter given. For strings, it checks that the string length is
			exactly that number of characters. For slices, arrays, and maps,
			validates the number of items.

		There doesn't appear to be a JSON schema equivalent to Length() for
		string types. Thus, using this with strings should result in the
		equivalent of both a Min() and Max() returning the same value.
	*/
	Length() float64

	/*
		Validator Package Documentation:

			For numbers, max will ensure that the value is less than or equal to
			the parameter given. For strings, it checks that the string length
			is at most that number of characters. For slices, arrays, and maps,
			validates the number of items.

		JSON Schema Validation RFC:

			6.2. maximum
				The value of "maximum" MUST be a number, representing an inclusive upper limit for a numeric instance.
				If the instance is a number, then this keyword validates only if the instance is less than or exactly equal to "maximum".
			6.6. maxLength
				The value of this keyword MUST be a non-negative integer.
				A string instance is valid against this keyword if its length is less than, or equal to, the value of this keyword.
				The length of a string instance is defined as the number of its characters as defined by RFC 7159 [RFC7159].
			6.11. maxItems
				The value of this keyword MUST be a non-negative integer.
				An array instance is valid against "maxItems" if its size is less than, or equal to, the value of this keyword.

		Returns -1 when this validation is not enforced.
	*/
	Max() float64

	/*
		Validator Package Documentation:

			For numbers, min will ensure that the value is greater or equal to
			the parameter given. For strings, it checks that the string length
			is at least that number of characters. For slices, arrays, and maps,
			validates the number of items.

		JSON Schema Validation RFC:

			6.4. minimum
				The value of "minimum" MUST be a number, representing an inclusive upper limit for a numeric instance.
				If the instance is a number, then this keyword validates only if the instance is greater than or exactly equal to "minimum".
			6.7. minLength
				The value of this keyword MUST be a non-negative integer.
				A string instance is valid against this keyword if its length is greater than, or equal to, the value of this keyword.
				The length of a string instance is defined as the number of its characters as defined by RFC 7159 [RFC7159].
				Omitting this keyword has the same behavior as a value of 0.
			6.12. minItems
				The value of this keyword MUST be a non-negative integer.
				An array instance is valid against "minItems" if its size is greater than, or equal to, the value of this keyword.
				Omitting this keyword has the same behavior as a value of 0.

		Returns -1 when this validation is not enforced.
	*/
	Min() float64

	/*
		Validator Package Documentation:

			For strings & numbers, eq will ensure that the value is equal to the
			parameter given. For slices, arrays, and maps, validates the number
			of items.

		The closest JSON schema equivalent to Equals() for string types is the
		regex pattern. Consequently, be careful what you equals for. For numeric
		values, using this should have the equivalent result of setting Min()
		and Max() with the same value.

		The returned boolean indicates if the validation is present from the
		source. A string is returned containing the raw value from the source.
		No guarantees are made about the parsability of the returned value in
		the case of numeric and plural types.
	*/
	Equals() (string, bool)

	/*
		Validator Package Documentation:

			For strings & numbers, this will ensure that the value is not equal
			to the parameter given. For slices, arrays, and maps, validates the
			number of items.

		There doesn't appear to be a JSON schema equivalent to NotEqual(). I
		don't know of a way to emulate the behavior without a complex boolean
		expression. Therefore, this method will not be implemented.
	*/
	// NotEqual() string

	/*
		Validator Package Documentation:

			For numbers, this will ensure that the value is greater than the
			parameter given. For strings, it checks that the string length is
			greater than that number of characters. For slices, arrays and maps
			it validates the number of items.

		JSON Schema Validation RFC:

			6.5. exclusiveMinimum
				The value of "exclusiveMinimum" MUST be number, representing an exclusive upper limit for a numeric instance.
				If the instance is a number, then the instance is valid only if it has a value strictly greater than (not equal to) "exclusiveMinimum".
			6.7. minLength
				The value of this keyword MUST be a non-negative integer.
				A string instance is valid against this keyword if its length is greater than, or equal to, the value of this keyword.
				The length of a string instance is defined as the number of its characters as defined by RFC 7159 [RFC7159].
				Omitting this keyword has the same behavior as a value of 0.
			6.12. minItems
				The value of this keyword MUST be a non-negative integer.
				An array instance is valid against "minItems" if its size is greater than, or equal to, the value of this keyword.
				Omitting this keyword has the same behavior as a value of 0.

		Returns -1 when this validation is not enforced.
	*/
	GreaterThan() float64

	/*
		Validator Package Documentation:

			For numbers, this will ensure that the value is less than the
			parameter given. For strings, it checks that the string length is
			less than that number of characters. For slices, arrays, and maps it
			validates the number of items.

		JSON Schema Validation RFC:

			6.3. exclusiveMaximum
				The value of "exclusiveMaximum" MUST be number, representing an exclusive upper limit for a numeric instance.
				If the instance is a number, then the instance is valid only if it has a value strictly less than (not equal to) "exclusiveMaximum".
			6.6. maxLength
				The value of this keyword MUST be a non-negative integer.
				A string instance is valid against this keyword if its length is less than, or equal to, the value of this keyword.
				The length of a string instance is defined as the number of its characters as defined by RFC 7159 [RFC7159].
			6.11. maxItems
				The value of this keyword MUST be a non-negative integer.
				An array instance is valid against "maxItems" if its size is less than, or equal to, the value of this keyword.

		Returns -1 when this validation is not enforced.
	*/
	LessThan() float64

	// The following are redundant, and their equivalent expressions in the
	// Validator package will be interpreted and returned with the Min() and
	// Max() accessors.
	//
	// GreaterThanOrEqual() string // Same as 'min' above. Kept both to make terminology with 'len' easier.
	// LessThanOrEqual() string // Same as 'max' above. Kept both to make terminology with 'len' easier.
}
