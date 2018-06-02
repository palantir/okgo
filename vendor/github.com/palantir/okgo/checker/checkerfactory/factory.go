// Copyright 2016 Palantir Technologies, Inc.
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

package checkerfactory

import (
	"sort"

	"github.com/pkg/errors"

	"github.com/palantir/okgo/checker"
	"github.com/palantir/okgo/okgo"
)

type checkerFactoryImpl struct {
	types                  []okgo.CheckerType
	checkerCreators        map[okgo.CheckerType]checker.CreatorFunction
	checkerConfigUpgraders map[okgo.CheckerType]okgo.ConfigUpgrader
}

func (f *checkerFactoryImpl) Types() []okgo.CheckerType {
	return f.types
}

func (f *checkerFactoryImpl) NewChecker(checkerType okgo.CheckerType, cfgYMLBytes []byte) (okgo.Checker, error) {
	creatorFn, ok := f.checkerCreators[checkerType]
	if !ok {
		return nil, errors.Errorf("no checker registered for checker type %q (registered checkers: %v)", checkerType, f.types)
	}
	return creatorFn(cfgYMLBytes)
}

func (f *checkerFactoryImpl) ConfigUpgrader(typeName okgo.CheckerType) (okgo.ConfigUpgrader, error) {
	if _, ok := f.checkerCreators[typeName]; !ok {
		return nil, errors.Errorf("check %q not registered (registered checks: %v)", typeName, f.types)
	}
	upgrader, ok := f.checkerConfigUpgraders[typeName]
	if !ok {
		return nil, errors.Errorf("%s is a valid formatter but does not have a config upgrader", typeName)
	}
	return upgrader, nil
}

func New(providedCheckerCreators []checker.Creator, providedConfigUpgraders []okgo.ConfigUpgrader) (okgo.CheckerFactory, error) {
	checkerCreators := make(map[okgo.CheckerType]checker.CreatorFunction)
	var checkers []okgo.CheckerType
	for _, currCreator := range providedCheckerCreators {
		checkerCreators[currCreator.Type()] = currCreator.Creator()
		checkers = append(checkers, currCreator.Type())
	}
	sort.Sort(okgo.ByCheckerType(checkers))
	configUpgraders := make(map[okgo.CheckerType]okgo.ConfigUpgrader)
	for _, currUpgrader := range providedConfigUpgraders {
		currUpgrader := currUpgrader
		configUpgraders[currUpgrader.TypeName()] = currUpgrader
	}
	return &checkerFactoryImpl{
		types:                  checkers,
		checkerCreators:        checkerCreators,
		checkerConfigUpgraders: configUpgraders,
	}, nil
}
