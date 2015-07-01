/*
Rating system designed to be used in VoIP Carriers World
Copyright (C) 2012-2015 ITsysCOM

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package engine

import (
	"bytes"
	"encoding/gob"
	"encoding/json"

	"github.com/cgrates/cgrates/utils"
	"gopkg.in/mgo.v2/bson"
	msgpack "gopkg.in/vmihailenco/msgpack.v2"
)

type Storage interface {
	Close()
	Flush(string) error
	GetKeysForPrefix(string) ([]string, error)
}

// Interface for storage providers.
type RatingStorage interface {
	Storage
	CacheAll() error
	CachePrefixes(...string) error
	CachePrefixValues(map[string][]string) error
	Cache([]string, []string, []string, []string, []string, []string, []string, []string, []string) error
	HasData(string, string) (bool, error)
	GetRatingPlan(string, bool) (*RatingPlan, error)
	SetRatingPlan(*RatingPlan) error
	GetRatingProfile(string, bool) (*RatingProfile, error)
	SetRatingProfile(*RatingProfile) error
	GetRpAlias(string, bool) (string, error)
	SetRpAlias(string, string) error
	RemoveRpAliases([]*TenantRatingSubject) error
	GetRPAliases(string, string, bool) ([]string, error)
	GetDestination(string) (*Destination, error)
	SetDestination(*Destination) error
	GetLCR(string, bool) (*LCR, error)
	SetLCR(*LCR) error
	SetCdrStats(*CdrStats) error
	GetCdrStats(string) (*CdrStats, error)
	GetAllCdrStats() ([]*CdrStats, error)
	GetDerivedChargers(string, bool) (utils.DerivedChargers, error)
	SetDerivedChargers(string, utils.DerivedChargers) error
	GetActions(string, bool) (Actions, error)
	SetActions(string, Actions) error
	GetSharedGroup(string, bool) (*SharedGroup, error)
	SetSharedGroup(*SharedGroup) error
	GetActionPlans(string) (ActionPlans, error)
	SetActionPlans(string, ActionPlans) error
	GetAllActionPlans() (map[string]ActionPlans, error)
	GetAccAlias(string, bool) (string, error)
	SetAccAlias(string, string) error
	RemoveAccAliases([]*TenantAccount) error
	GetAccountAliases(string, string, bool) ([]string, error)
}

type AccountingStorage interface {
	Storage
	GetAccount(string) (*Account, error)
	SetAccount(*Account) error
	GetCdrStatsQueue(string) (*StatsQueue, error)
	SetCdrStatsQueue(*StatsQueue) error
}

type CdrStorage interface {
	Storage
	SetCdr(*StoredCdr) error
	SetRatedCdr(*StoredCdr) error
	LogCallCost(cgrid, source, runid string, cc *CallCost) error
	GetCallCostLog(cgrid, source, runid string) (*CallCost, error)
	GetStoredCdrs(*utils.CdrsFilter) ([]*StoredCdr, int64, error)
	RemStoredCdrs([]string) error
}

type LogStorage interface {
	Storage
	//GetAllActionTimingsLogs() (map[string]ActionsTimings, error)
	LogError(uuid, source, runid, errstr string) error
	LogActionTrigger(ubId, source string, at *ActionTrigger, as Actions) error
	LogActionPlan(source string, at *ActionPlan, as Actions) error
}

type LoadStorage interface {
	Storage
	LoadReader
	LoadWriter
}

type LoadReader interface {
	GetTpIds() ([]string, error)
	GetTpTableIds(string, string, utils.TPDistinctIds, map[string]string, *utils.Paginator) ([]string, error)
	GetTpTimings(string, string) ([]TpTiming, error)
	GetTpDestinations(string, string) ([]TpDestination, error)
	GetTpRates(string, string) ([]TpRate, error)
	GetTpDestinationRates(string, string, *utils.Paginator) ([]TpDestinationRate, error)
	GetTpRatingPlans(string, string, *utils.Paginator) ([]TpRatingPlan, error)
	GetTpRatingProfiles(*TpRatingProfile) ([]TpRatingProfile, error)
	GetTpSharedGroups(string, string) ([]TpSharedGroup, error)
	GetTpCdrStats(string, string) ([]TpCdrstat, error)
	GetTpDerivedChargers(*TpDerivedCharger) ([]TpDerivedCharger, error)
	GetTpLCRs(string, string) ([]TpLcrRule, error)
	GetTpActions(string, string) ([]TpAction, error)
	GetTpActionPlans(string, string) ([]TpActionPlan, error)
	GetTpActionTriggers(string, string) ([]TpActionTrigger, error)
	GetTpAccountActions(*TpAccountAction) ([]TpAccountAction, error)
}

type LoadWriter interface {
	RemTpData(string, string, ...string) error
	SetTpTimings([]TpTiming) error
	SetTpDestinations([]TpDestination) error
	SetTpRates([]TpRate) error
	SetTpDestinationRates([]TpDestinationRate) error
	SetTpRatingPlans([]TpRatingPlan) error
	SetTpRatingProfiles([]TpRatingProfile) error
	SetTpSharedGroups([]TpSharedGroup) error
	SetTpCdrStats([]TpCdrstat) error
	SetTpDerivedChargers([]TpDerivedCharger) error
	SetTpLCRs([]TpLcrRule) error
	SetTpActions([]TpAction) error
	SetTpActionPlans([]TpActionPlan) error
	SetTpActionTriggers([]TpActionTrigger) error
	SetTpAccountActions([]TpAccountAction) error
}

type Marshaler interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

type JSONMarshaler struct{}

func (jm *JSONMarshaler) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (jm *JSONMarshaler) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

type BSONMarshaler struct{}

func (jm *BSONMarshaler) Marshal(v interface{}) ([]byte, error) {
	return bson.Marshal(v)
}

func (jm *BSONMarshaler) Unmarshal(data []byte, v interface{}) error {
	return bson.Unmarshal(data, v)
}

type JSONBufMarshaler struct{}

func (jbm *JSONBufMarshaler) Marshal(v interface{}) (data []byte, err error) {
	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(v)
	data = buf.Bytes()
	return
}

func (jbm *JSONBufMarshaler) Unmarshal(data []byte, v interface{}) error {
	return json.NewDecoder(bytes.NewBuffer(data)).Decode(v)
}

/*
type CodecMsgpackMarshaler struct {
	mh *codec.MsgpackHandle
}

func NewCodecMsgpackMarshaler() *CodecMsgpackMarshaler {
	cmm := &CodecMsgpackMarshaler{new(codec.MsgpackHandle)}
	mh := cmm.mh
	mh.MapType = reflect.TypeOf(map[string]interface{}(nil))
	return cmm
}

func (cmm *CodecMsgpackMarshaler) Marshal(v interface{}) (b []byte, err error) {
	enc := codec.NewEncoderBytes(&b, cmm.mh)
	err = enc.Encode(v)
	return
}

func (cmm *CodecMsgpackMarshaler) Unmarshal(data []byte, v interface{}) error {
	dec := codec.NewDecoderBytes(data, cmm.mh)
	return dec.Decode(&v)
}

type BincMarshaler struct {
	bh *codec.BincHandle
}

func NewBincMarshaler() *BincMarshaler {
	return &BincMarshaler{new(codec.BincHandle)}
}

func (bm *BincMarshaler) Marshal(v interface{}) (b []byte, err error) {
	enc := codec.NewEncoderBytes(&b, bm.bh)
	err = enc.Encode(v)
	return
}

func (bm *BincMarshaler) Unmarshal(data []byte, v interface{}) error {
	dec := codec.NewDecoderBytes(data, bm.bh)
	return dec.Decode(&v)
}*/

type GOBMarshaler struct{}

func (gm *GOBMarshaler) Marshal(v interface{}) (data []byte, err error) {
	buf := new(bytes.Buffer)
	err = gob.NewEncoder(buf).Encode(v)
	data = buf.Bytes()
	return
}

func (gm *GOBMarshaler) Unmarshal(data []byte, v interface{}) error {
	return gob.NewDecoder(bytes.NewBuffer(data)).Decode(v)
}

type MsgpackMarshaler struct{}

func (jm *MsgpackMarshaler) Marshal(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (jm *MsgpackMarshaler) Unmarshal(data []byte, v interface{}) error {
	return msgpack.Unmarshal(data, v)
}
