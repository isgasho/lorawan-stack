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

package store

import (
	"context"
	"reflect"

	"github.com/gogo/protobuf/types"
	"github.com/jinzhu/gorm"
	"go.thethings.network/lorawan-stack/pkg/ttnpb"
)

// GetOrganizationStore returns an OrganizationStore on the given db (or transaction).
func GetOrganizationStore(db *gorm.DB) OrganizationStore {
	return &organizationStore{db: db}
}

type organizationStore struct {
	db *gorm.DB
}

// selectOrganizationFields selects relevant fields (based on fieldMask) and preloads details if needed.
func selectOrganizationFields(query *gorm.DB, fieldMask *types.FieldMask) *gorm.DB {
	if fieldMask == nil || len(fieldMask.Paths) == 0 {
		return query.Preload("Attributes").Preload("ContactInfo").Select([]string{"organizations.*", "accounts.uid"})
	}
	var organizationColumns []string
	for _, column := range modelColumns {
		organizationColumns = append(organizationColumns, "organizations."+column)
	}
	for _, path := range fieldMask.Paths {
		switch path {
		case "ids.organization_id":
			// accounts.uid is always selected
		case attributesField:
			query = query.Preload("Attributes")
		case contactInfoField:
			query = query.Preload("ContactInfo")
		default:
			if column, ok := organizationColumnNames[path]; ok {
				organizationColumns = append(organizationColumns, column)
			} else {
				organizationColumns = append(organizationColumns, path)
			}
		}
	}
	return query.Select(append(organizationColumns, "accounts.uid"))
}

func (s *organizationStore) CreateOrganization(ctx context.Context, org *ttnpb.Organization) (*ttnpb.Organization, error) {
	orgModel := Organization{
		Account: Account{UID: org.OrganizationID}, // The ID is not mutated by fromPB.
	}
	orgModel.fromPB(org, nil)
	orgModel.SetContext(ctx)
	query := s.db.Create(&orgModel)
	if query.Error != nil {
		return nil, query.Error
	}
	var orgProto ttnpb.Organization
	orgModel.toPB(&orgProto, nil)
	return &orgProto, nil
}

func (s *organizationStore) FindOrganizations(ctx context.Context, ids []*ttnpb.OrganizationIdentifiers, fieldMask *types.FieldMask) ([]*ttnpb.Organization, error) {
	idStrings := make([]string, len(ids))
	for i, id := range ids {
		idStrings[i] = id.GetOrganizationID()
	}
	query := s.db.Scopes(withContext(ctx), withOrganizationID(idStrings...))
	query = selectOrganizationFields(query, fieldMask)
	if limit, offset := limitAndOffsetFromContext(ctx); limit != 0 {
		countTotal(ctx, query.Model(&Organization{}))
		query = query.Limit(limit).Offset(offset)
	}
	var orgModels []Organization
	query = query.Find(&orgModels)
	setTotal(ctx, uint64(len(orgModels)))
	if query.Error != nil {
		return nil, query.Error
	}
	orgProtos := make([]*ttnpb.Organization, len(orgModels))
	for i, orgModel := range orgModels {
		orgProto := new(ttnpb.Organization)
		orgModel.toPB(orgProto, fieldMask)
		orgProtos[i] = orgProto
	}
	return orgProtos, nil
}

func (s *organizationStore) GetOrganization(ctx context.Context, id *ttnpb.OrganizationIdentifiers, fieldMask *types.FieldMask) (*ttnpb.Organization, error) {
	query := s.db.Scopes(withContext(ctx), withOrganizationID(id.GetOrganizationID()))
	query = selectOrganizationFields(query, fieldMask)
	var orgModel Organization
	err := query.Preload("Account").First(&orgModel).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, errNotFoundForID(id.EntityIdentifiers())
		}
		return nil, err
	}
	orgProto := new(ttnpb.Organization)
	orgModel.toPB(orgProto, fieldMask)
	return orgProto, nil
}

func (s *organizationStore) UpdateOrganization(ctx context.Context, org *ttnpb.Organization, fieldMask *types.FieldMask) (updated *ttnpb.Organization, err error) {
	query := s.db.Scopes(withContext(ctx), withOrganizationID(org.GetOrganizationID()))
	query = selectOrganizationFields(query, fieldMask)
	var orgModel Organization
	err = query.Preload("Account").First(&orgModel).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, errNotFoundForID(org.OrganizationIdentifiers.EntityIdentifiers())
		}
		return nil, err
	}
	if !org.UpdatedAt.IsZero() && org.UpdatedAt != orgModel.UpdatedAt {
		return nil, errConcurrentWrite
	}
	if err := ctx.Err(); err != nil { // Early exit if context canceled
		return nil, err
	}
	oldAttributes, oldContactInfo := orgModel.Attributes, orgModel.ContactInfo
	columns := orgModel.fromPB(org, fieldMask)
	if len(columns) > 0 {
		query = s.db.Model(&orgModel).Select(columns).Updates(&orgModel)
		if query.Error != nil {
			return nil, query.Error
		}
	}
	if !reflect.DeepEqual(oldAttributes, orgModel.Attributes) {
		err = replaceAttributes(s.db, "organization", orgModel.ID, oldAttributes, orgModel.Attributes)
		if err != nil {
			return nil, err
		}
	}
	if !reflect.DeepEqual(oldContactInfo, orgModel.ContactInfo) {
		err = replaceContactInfos(s.db, "organization", orgModel.ID, oldContactInfo, orgModel.ContactInfo)
		if err != nil {
			return nil, err
		}
	}
	updated = new(ttnpb.Organization)
	orgModel.toPB(updated, fieldMask)
	return updated, nil
}

func (s *organizationStore) DeleteOrganization(ctx context.Context, id *ttnpb.OrganizationIdentifiers) (err error) {
	defer func() {
		if err != nil && gorm.IsRecordNotFoundError(err) {
			err = errNotFoundForID(id.EntityIdentifiers())
		}
	}()
	query := s.db.Scopes(withContext(ctx), withOrganizationID(id.GetOrganizationID()))
	query = query.Select("organizations.id")
	var orgModel Organization
	err = query.First(&orgModel).Error
	if err != nil {
		return err
	}
	return s.db.Delete(&orgModel).Error
}
