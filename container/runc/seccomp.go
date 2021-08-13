/*
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/
package runc

import (
	"errors"
	"fmt"
)

type Seccomp struct {
	DefaultAction   Action     `json:"default_action"`
	Architectures   []string   `json:"architectures"`
	Syscalls        []*Syscall `json:"syscalls"`
	DefaultErrnoRet *uint      `json:"default_errno_ret"`
}

type Operator int

// Arg is a rule to match a specific syscall argument in Seccomp
type Arg struct {
	Index    uint     `json:"index"`
	Value    uint64   `json:"value"`
	ValueTwo uint64   `json:"value_two"`
	Op       Operator `json:"op"`
}

// Syscall is a rule to match a syscall in Seccomp
type Syscall struct {
	Name     string `json:"name"`
	Action   Action `json:"action"`
	ErrnoRet *uint  `json:"errnoRet"`
	Args     []*Arg `json:"args"`
}

type Action int

// Action is taken
func SetupSeccomp(config *LinuxSeccomp) (*Seccomp, error) {
	if config == nil {
		return nil, nil
	}

	// No default action specified, no syscalls listed, assume seccomp disabled
	if config.DefaultAction == "" && len(config.Syscalls) == 0 {
		return nil, nil
	}

	// We don't currently support seccomp flags.
	if len(config.Flags) != 0 {
		return nil, errors.New("seccomp flags are not yet supported by runc")
	}

	newConfig := new(Seccomp)
	newConfig.Syscalls = []*Syscall{}

	if len(config.Architectures) > 0 {
		newConfig.Architectures = []string{}
		for _, arch := range config.Architectures {
			newArch, err := ConvertStringToArch(string(arch))
			if err != nil {
				return nil, err
			}
			newConfig.Architectures = append(newConfig.Architectures, newArch)
		}
	}

	// Convert default action from string representation
	newDefaultAction, err := ConvertStringToAction(string(config.DefaultAction))
	if err != nil {
		return nil, err
	}
	newConfig.DefaultAction = newDefaultAction
	newConfig.DefaultErrnoRet = config.DefaultErrnoRet

	// Loop through all syscall blocks and convert them to libcontainer format
	for _, call := range config.Syscalls {
		newAction, err := ConvertStringToAction(string(call.Action))
		if err != nil {
			return nil, err
		}

		for _, name := range call.Names {
			newCall := Syscall{
				Name:     name,
				Action:   newAction,
				ErrnoRet: call.ErrnoRet,
				Args:     []*Arg{},
			}
			// Loop through all the arguments of the syscall and convert them
			for _, arg := range call.Args {
				newOp, err := ConvertStringToOperator(string(arg.Op))
				if err != nil {
					return nil, err
				}

				newArg := Arg{
					Index:    arg.Index,
					Value:    arg.Value,
					ValueTwo: arg.ValueTwo,
					Op:       newOp,
				}

				newCall.Args = append(newCall.Args, &newArg)
			}
			newConfig.Syscalls = append(newConfig.Syscalls, &newCall)
		}
	}

	return newConfig, nil
}

// ConvertStringToArch converts a string into a Seccomp comparison arch.
func ConvertStringToArch(in string) (string, error) {
	if arch, ok := archs[in]; ok {
		return arch, nil
	}
	return "", fmt.Errorf("string %s is not a valid arch for seccomp", in)
}

func ConvertStringToAction(in string) (Action, error) {
	if act, ok := actions[in]; ok {
		return act, nil
	}
	return 0, fmt.Errorf("string %s is not a valid action for seccomp", in)
}

var operators = map[string]Operator{
	"SCMP_CMP_NE":        NotEqualTo,
	"SCMP_CMP_LT":        LessThan,
	"SCMP_CMP_LE":        LessThanOrEqualTo,
	"SCMP_CMP_EQ":        EqualTo,
	"SCMP_CMP_GE":        GreaterThanOrEqualTo,
	"SCMP_CMP_GT":        GreaterThan,
	"SCMP_CMP_MASKED_EQ": MaskEqualTo,
}

var actions = map[string]Action{
	"SCMP_ACT_KILL":  Kill,
	"SCMP_ACT_ERRNO": Errno,
	"SCMP_ACT_TRAP":  Trap,
	"SCMP_ACT_ALLOW": Allow,
	"SCMP_ACT_TRACE": Trace,
	"SCMP_ACT_LOG":   Log,
}

var archs = map[string]string{
	"SCMP_ARCH_X86":         "x86",
	"SCMP_ARCH_X86_64":      "amd64",
	"SCMP_ARCH_X32":         "x32",
	"SCMP_ARCH_ARM":         "arm",
	"SCMP_ARCH_AARCH64":     "arm64",
	"SCMP_ARCH_MIPS":        "mips",
	"SCMP_ARCH_MIPS64":      "mips64",
	"SCMP_ARCH_MIPS64N32":   "mips64n32",
	"SCMP_ARCH_MIPSEL":      "mipsel",
	"SCMP_ARCH_MIPSEL64":    "mipsel64",
	"SCMP_ARCH_MIPSEL64N32": "mipsel64n32",
	"SCMP_ARCH_PPC":         "ppc",
	"SCMP_ARCH_PPC64":       "ppc64",
	"SCMP_ARCH_PPC64LE":     "ppc64le",
	"SCMP_ARCH_S390":        "s390",
	"SCMP_ARCH_S390X":       "s390x",
}

func ConvertStringToOperator(in string) (Operator, error) {
	if op, ok := operators[in]; ok {
		return op, nil
	}
	return 0, fmt.Errorf("string %s is not a valid operator for seccomp", in)
}

const (
	EqualTo Operator = iota + 1
	NotEqualTo
	GreaterThan
	GreaterThanOrEqualTo
	LessThan
	LessThanOrEqualTo
	MaskEqualTo
)

const (
	Kill Action = iota + 1
	Errno
	Trap
	Allow
	Trace
	Log
)
