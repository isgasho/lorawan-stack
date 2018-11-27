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
	"github.com/gogo/protobuf/types"
	"github.com/lib/pq"
	"go.thethings.network/lorawan-stack/pkg/ttnpb"
)

// Client model.
type Client struct {
	Model
	SoftDelete

	// BEGIN common fields
	ClientID    string        `gorm:"unique_index:id;type:VARCHAR(36)"`
	Name        string        `gorm:"type:VARCHAR"`
	Description string        `gorm:"type:TEXT"`
	Attributes  []Attribute   `gorm:"polymorphic:Entity;polymorphic_value:client"`
	ContactInfo []ContactInfo `gorm:"polymorphic:Entity;polymorphic_value:client"`
	Memberships []Membership  `gorm:"polymorphic:Entity;polymorphic_value:client"`
	// END common fields

	ClientSecret string         `gorm:"type:VARCHAR"`
	RedirectURIs pq.StringArray `gorm:"type:VARCHAR ARRAY;column:redirect_uris"`

	State int `gorm:"not null"`

	SkipAuthorization bool
	Endorsed          bool

	Grants Grants `gorm:"type:INT ARRAY"`
	Rights Rights `gorm:"type:INT ARRAY"`
}

func init() {
	registerModel(&Client{})
}

// functions to set fields from the client model into the client proto.
var clientPBSetters = map[string]func(*ttnpb.Client, *Client){
	nameField:              func(pb *ttnpb.Client, cli *Client) { pb.Name = cli.Name },
	descriptionField:       func(pb *ttnpb.Client, cli *Client) { pb.Description = cli.Description },
	attributesField:        func(pb *ttnpb.Client, cli *Client) { pb.Attributes = attributes(cli.Attributes).toMap() },
	contactInfoField:       func(pb *ttnpb.Client, cli *Client) { pb.ContactInfo = contactInfos(cli.ContactInfo).toPB() },
	secretField:            func(pb *ttnpb.Client, cli *Client) { pb.Secret = cli.ClientSecret },
	redirectURIsField:      func(pb *ttnpb.Client, cli *Client) { pb.RedirectURIs = cli.RedirectURIs },
	stateField:             func(pb *ttnpb.Client, cli *Client) { pb.State = ttnpb.State(cli.State) },
	skipAuthorizationField: func(pb *ttnpb.Client, cli *Client) { pb.SkipAuthorization = cli.SkipAuthorization },
	endorsedField:          func(pb *ttnpb.Client, cli *Client) { pb.Endorsed = cli.Endorsed },
	grantsField:            func(pb *ttnpb.Client, cli *Client) { pb.Grants = cli.Grants },
	rightsField:            func(pb *ttnpb.Client, cli *Client) { pb.Rights = cli.Rights.Rights },
}

// functions to set fields from the client proto into the client model.
var clientModelSetters = map[string]func(*Client, *ttnpb.Client){
	nameField:        func(cli *Client, pb *ttnpb.Client) { cli.Name = pb.Name },
	descriptionField: func(cli *Client, pb *ttnpb.Client) { cli.Description = pb.Description },
	attributesField: func(cli *Client, pb *ttnpb.Client) {
		cli.Attributes = attributes(cli.Attributes).updateFromMap(pb.Attributes)
	},
	contactInfoField: func(cli *Client, pb *ttnpb.Client) {
		cli.ContactInfo = contactInfos(cli.ContactInfo).updateFromPB(pb.ContactInfo)
	},
	secretField:            func(cli *Client, pb *ttnpb.Client) { cli.ClientSecret = pb.Secret },
	redirectURIsField:      func(cli *Client, pb *ttnpb.Client) { cli.RedirectURIs = pq.StringArray(pb.RedirectURIs) },
	stateField:             func(cli *Client, pb *ttnpb.Client) { cli.State = int(pb.State) },
	skipAuthorizationField: func(cli *Client, pb *ttnpb.Client) { cli.SkipAuthorization = pb.SkipAuthorization },
	endorsedField:          func(cli *Client, pb *ttnpb.Client) { cli.Endorsed = pb.Endorsed },
	grantsField:            func(cli *Client, pb *ttnpb.Client) { cli.Grants = pb.Grants },
	rightsField:            func(cli *Client, pb *ttnpb.Client) { cli.Rights = Rights{Rights: pb.Rights} },
}

// fieldMask to use if a nil or empty fieldmask is passed.
var defaultClientFieldMask = &types.FieldMask{}

func init() {
	paths := make([]string, 0, len(clientPBSetters))
	for path := range clientPBSetters {
		paths = append(paths, path)
	}
	defaultClientFieldMask.Paths = paths
}

// fieldmask path to column name in clients table, if other than proto field.
var clientColumnNames = map[string]string{
	"ids.client_id":  "client_id",
	attributesField:  "Attributes",
	contactInfoField: "ContactInfo",
	secretField:      "client_secret",
}

func (cli Client) toPB(pb *ttnpb.Client, fieldMask *types.FieldMask) {
	pb.ClientIdentifiers.ClientID = cli.ClientID
	pb.CreatedAt = cleanTime(cli.CreatedAt)
	pb.UpdatedAt = cleanTime(cli.UpdatedAt)
	if fieldMask == nil || len(fieldMask.Paths) == 0 {
		fieldMask = defaultClientFieldMask
	}
	for _, path := range fieldMask.Paths {
		if setter, ok := clientPBSetters[path]; ok {
			setter(pb, &cli)
		}
	}
}

func (cli *Client) fromPB(pb *ttnpb.Client, fieldMask *types.FieldMask) (columns []string) {
	if fieldMask == nil || len(fieldMask.Paths) == 0 {
		fieldMask = defaultClientFieldMask
	}
	for _, path := range fieldMask.Paths {
		if setter, ok := clientModelSetters[path]; ok {
			setter(cli, pb)
			columnName, ok := clientColumnNames[path]
			if !ok {
				columnName = path
			}
			if columnName != "" {
				columns = append(columns, columnName)
			}
			continue
		}
	}
	return
}