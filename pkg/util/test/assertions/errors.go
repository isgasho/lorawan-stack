// Copyright © 2018 The Things Network Foundation, The Things Industries B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package assertions

import (
	"fmt"
	"reflect"

	removetheseerrors "go.thethings.network/lorawan-stack/pkg/errors"
	errors "go.thethings.network/lorawan-stack/pkg/errorsv3"
)

const (
	needDescriptor                = "This assertion requires ErrDescriptor as comparison type (you provided %T)."
	shouldBeErrorType             = "Expected a known error value (but was of type %T instead)!"
	shouldHaveNamespace           = "Expected error to have namespace '%v' (but it was '%v' instead)!"
	shouldHaveCode                = "Expected error to have code '%v' (but it was '%v' instead)!"
	shouldNotDescribe             = "Expected error to not describe '%v' (but it does)!"
	needDefinitionCompatible      = "This assertion requires a Definition-compatible comparison type (you provided %T)."
	needErrorDefinitionCompatible = "This assertion requires an Error-compatible or Definition-compatible comparison type (you provided %T)."
	shouldBeErrorCompatible       = "Expected an Error-compatible value (but was of type %T instead)!"
	shouldBeDefinitionCompatible  = "Expected a Definition-compatible value (but was of type %T instead)!"
	shouldHaveName                = "Expected error to have name '%v' (but it was '%v' instead)!"
	shouldHaveMessageFormat       = "Expected error to have message format '%v' (but it was '%v' instead)!"
	shouldHaveAttributes          = "Expected error to have attributes '%v' (but it was '%v' instead)!"
	shouldHaveCause               = "Expected error to have cause '%v' (but it was '%v' instead)!"
	shouldHaveDetails             = "Expected error to have details '%v' (but it was '%v' instead)!"
)

// ShouldDescribeError takes as argument an Error and an ErrDescriptor. If the arguments are
// valid and the Error was generated by this ErrDescriptor, the function returns an
// empty string. Otherwise, it returns a string describing the error.
func ShouldDescribeError(actual interface{}, expected ...interface{}) string {
	descriptor, ok := expected[0].(*removetheseerrors.ErrDescriptor)
	if !ok {
		return fmt.Sprintf(needDescriptor, expected[0])
	}

	err, ok := actual.(error)
	if !ok {
		return fmt.Sprintf(shouldBeErrorType, actual)
	}
	if message := need(1, expected); message != success {
		return message
	}

	if !descriptor.Describes(err) {
		err := removetheseerrors.From(err)
		if err.Namespace() != descriptor.Namespace {
			return fmt.Sprintf(shouldHaveNamespace, descriptor.Namespace, err.Namespace())
		}
		if err.Code() != descriptor.Code {
			return fmt.Sprintf(shouldHaveCode, descriptor.Code, err.Code())
		}
	}

	return success
}

// ShouldNotDescribeError takes as argument an Error and an ErrDescriptor. If the Error was
// not generated by this ErrDescriptor, the function returns an empty string. If the
// arguments are invalid or the Error was generated by this ErrDescriptor, the function
// returns a non-empty string.
func ShouldNotDescribeError(actual interface{}, expected ...interface{}) string {
	if message := need(1, expected); message != success {
		return message
	}
	if ShouldDescribeError(actual, expected...) == success {
		return fmt.Sprintf(shouldNotDescribe, expected[0])
	}
	return success
}

// ShouldHaveSameErrorDefinitionAs is used to assert that an error resembles the given Error or Definition.
func ShouldHaveSameErrorDefinitionAs(actual interface{}, expected ...interface{}) string {
	if len(expected) != 1 {
		return fmt.Sprintf(needExactValues, 1, len(expected))
	}
	expectedErr, ok := expected[0].(errors.DefinitionInterface)
	if !ok {
		return fmt.Sprintf(needDefinitionCompatible, expected[0])
	}
	actualErr, ok := actual.(errors.DefinitionInterface)
	if !ok {
		return fmt.Sprintf(shouldBeDefinitionCompatible, actual)
	}
	return assertDefinitionCompatibleEquals(actualErr, expectedErr)
}

func assertDefinitionCompatibleEquals(actual, expected errors.DefinitionInterface) string {
	if actual.Namespace() != expected.Namespace() {
		return fmt.Sprintf(shouldHaveNamespace, expected.Namespace(), actual.Namespace())
	}
	if actual.Name() != expected.Name() {
		return fmt.Sprintf(shouldHaveName, expected.Name(), actual.Name())
	}
	if actual.MessageFormat() != expected.MessageFormat() {
		return fmt.Sprintf(shouldHaveMessageFormat, expected.MessageFormat(), actual.MessageFormat())
	}
	if actual.Code() != expected.Code() {
		return fmt.Sprintf(shouldHaveCode, expected.Code(), actual.Code())
	}
	return success
}

func assertErrorCompatibleEquals(actual, expected errors.Interface) string {
	if assertDefinition := assertDefinitionCompatibleEquals(actual, expected); assertDefinition != success {
		return assertDefinition
	}
	if !reflect.DeepEqual(actual.Attributes(), expected.Attributes()) {
		return fmt.Sprintf(shouldHaveAttributes, expected.Attributes(), actual.Attributes())
	}
	if actual.Cause() != expected.Cause() {
		return fmt.Sprintf(shouldHaveCause, expected.Cause(), actual.Cause())
	}
	if !reflect.DeepEqual(actual.Details(), expected.Details()) {
		return fmt.Sprintf(shouldHaveDetails, expected.Details(), actual.Details())
	}
	return success
}

// ShouldEqualErrorOrDefinition is used to assert that an error equals the given Error or Definition.
func ShouldEqualErrorOrDefinition(actual interface{}, expected ...interface{}) string {
	if len(expected) != 1 {
		return fmt.Sprintf(needExactValues, 1, len(expected))
	}
	if actual == nil && expected[0] == nil {
		return success
	}
	if expected, ok := expected[0].(errors.Interface); ok {
		if actual, ok := actual.(errors.Interface); ok {
			return assertErrorCompatibleEquals(actual, expected)
		}
		return fmt.Sprintf(shouldBeErrorCompatible, actual)
	}
	if expected, ok := expected[0].(errors.DefinitionInterface); ok {
		if actual, ok := actual.(errors.DefinitionInterface); ok {
			return assertDefinitionCompatibleEquals(actual, expected)
		}
		return fmt.Sprintf(shouldBeDefinitionCompatible, actual)
	}
	return fmt.Sprintf(needErrorDefinitionCompatible, actual)
}
