/*
 * Copyright (c) Microsoft Corporation.
 * Licensed under the MIT license.
 * SPDX-License-Identifier: MIT
 */

package users

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/contexts"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/managers"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/observability"
	observ_utils "github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/observability/utils"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/providers"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/providers/states"
	"github.com/eclipse-symphony/symphony/coa/pkg/logger"
)

var log = logger.NewLogger("coa.runtime")

type UsersManager struct {
	managers.Manager
	StateProvider states.IStateProvider
}

type UserState struct {
	Id           string   `json:"id"`
	PasswordHash string   `json:"passwordHash,omitempty"`
	Roles        []string `json:"roles,omitempty"`
}

func (s *UsersManager) Init(context *contexts.VendorContext, config managers.ManagerConfig, providers map[string]providers.IProvider) error {
	stateprovider, err := managers.GetStateProvider(config, providers)
	if err == nil {
		s.StateProvider = stateprovider
	} else {
		return err
	}

	return nil
}
func (t *UsersManager) DeleteUser(ctx context.Context, name string) error {
	ctx, span := observability.StartSpan("Users Manager", ctx, &map[string]string{
		"method": "DeleteUser",
	})
	var err error = nil
	defer observ_utils.CloseSpanWithError(span, &err)

	err = t.StateProvider.Delete(ctx, states.DeleteRequest{
		ID: name,
	})
	return err
}

func hash(name string, s string) string {
	h := fnv.New32a()
	h.Write([]byte(name + "." + s + ".salt"))
	return fmt.Sprintf("H%d", h.Sum32())
}

func (t *UsersManager) UpsertUser(ctx context.Context, name string, password string, roles []string) error {
	ctx, span := observability.StartSpan("Users Manager", ctx, &map[string]string{
		"method": "UpsertUser",
	})
	var err error = nil
	defer observ_utils.CloseSpanWithError(span, &err)

	log.Debug(" M (Users) : upsert user")
	upsertRequest := states.UpsertRequest{
		Value: states.StateEntry{
			ID: name,
			Body: UserState{
				Id:           name,
				PasswordHash: hash(name, password),
				Roles:        roles,
			},
		},
	}
	_, err = t.StateProvider.Upsert(ctx, upsertRequest)
	if err != nil {
		log.Debugf(" M (Users) : failed to upsert user - %s", err)
		return err
	}
	return nil
}
func (t *UsersManager) CheckUser(ctx context.Context, name string, password string) ([]string, bool) {
	ctx, span := observability.StartSpan("Users Manager", ctx, &map[string]string{
		"method": "CheckUser",
	})
	var err error = nil
	defer observ_utils.CloseSpanWithError(span, &err)

	log.Debug(" M (Users) : check user")
	getRequest := states.GetRequest{
		ID: name,
	}
	user, err := t.StateProvider.Get(ctx, getRequest)
	if err != nil {
		log.Debugf(" M (Users) : failed to read user - %s", err)
		return nil, false
	}

	if v, ok := user.Body.(UserState); ok {
		if hash(name, password) == v.PasswordHash {
			log.Debug(" M (Users) : user authenticated")
			return v.Roles, true
		}
	}
	log.Debug(" M (Users) : authentication failed")
	return nil, false
}
