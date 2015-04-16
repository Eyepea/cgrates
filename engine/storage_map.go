/*
Real-time Charging System for Telecom & ISP environments
Copyright (C) 2012-2015 ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either vemsion 3 of the License, or
(at your option) any later vemsion.

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
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"strings"
	"time"

	"github.com/cgrates/cgrates/cache2go"
	"github.com/cgrates/cgrates/utils"
)

type MapStorage struct {
	dict map[string][]byte
	ms   Marshaler
}

func NewMapStorage() (*MapStorage, error) {
	return &MapStorage{dict: make(map[string][]byte), ms: NewCodecMsgpackMarshaler()}, nil
}

func NewMapStorageJson() (*MapStorage, error) {
	return &MapStorage{dict: make(map[string][]byte), ms: new(JSONBufMarshaler)}, nil
}

func (ms *MapStorage) Close() {}

func (ms *MapStorage) Flush(ignore string) error {
	ms.dict = make(map[string][]byte)
	return nil
}

func (ms *MapStorage) GetKeysForPrefix(prefix string) ([]string, error) {
	keysForPrefix := make([]string, 0)
	for key := range ms.dict {
		if strings.HasPrefix(key, prefix) {
			keysForPrefix = append(keysForPrefix, key)
		}
	}
	return keysForPrefix, nil
}

func (ms *MapStorage) CacheRating(dKeys, rpKeys, rpfKeys, alsKeys, lcrKeys []string) error {
	cache2go.BeginTransaction()
	if dKeys == nil || (float64(cache2go.CountEntries(DESTINATION_PREFIX))*DESTINATIONS_LOAD_THRESHOLD < float64(len(dKeys))) {
		cache2go.RemPrefixKey(DESTINATION_PREFIX)
	} else {
		CleanStalePrefixes(dKeys)
	}
	if rpKeys == nil {
		cache2go.RemPrefixKey(RATING_PLAN_PREFIX)
	}
	if rpfKeys == nil {
		cache2go.RemPrefixKey(RATING_PROFILE_PREFIX)
	}
	if alsKeys == nil {
		cache2go.RemPrefixKey(RP_ALIAS_PREFIX)
	}
	if lcrKeys == nil {
		cache2go.RemPrefixKey(LCR_PREFIX)
	}
	for k, _ := range ms.dict {
		if strings.HasPrefix(k, DESTINATION_PREFIX) {
			if _, err := ms.GetDestination(k[len(DESTINATION_PREFIX):]); err != nil {
				cache2go.RollbackTransaction()
				return err
			}
		}
		if strings.HasPrefix(k, RATING_PLAN_PREFIX) {
			cache2go.RemKey(k)
			if _, err := ms.GetRatingPlan(k[len(RATING_PLAN_PREFIX):], true); err != nil {
				cache2go.RollbackTransaction()
				return err
			}
		}
		if strings.HasPrefix(k, RATING_PROFILE_PREFIX) {
			cache2go.RemKey(k)
			if _, err := ms.GetRatingProfile(k[len(RATING_PROFILE_PREFIX):], true); err != nil {
				cache2go.RollbackTransaction()
				return err
			}
		}
		if strings.HasPrefix(k, RP_ALIAS_PREFIX) {
			cache2go.RemKey(k)
			if _, err := ms.GetRpAlias(k[len(RP_ALIAS_PREFIX):], true); err != nil {
				cache2go.RollbackTransaction()
				return err
			}
		}
		if strings.HasPrefix(k, LCR_PREFIX) {
			cache2go.RemKey(k)
			if _, err := ms.GetLCR(k[len(LCR_PREFIX):], true); err != nil {
				cache2go.RollbackTransaction()
				return err
			}
		}
	}
	cache2go.CommitTransaction()
	return nil
}

func (ms *MapStorage) CacheAccounting(actKeys, shgKeys, alsKeys, dcsKeys []string) error {
	cache2go.BeginTransaction()
	if actKeys == nil {
		cache2go.RemPrefixKey(ACTION_PREFIX) // Forced until we can fine tune it
	}
	if shgKeys == nil {
		cache2go.RemPrefixKey(SHARED_GROUP_PREFIX) // Forced until we can fine tune it
	}
	if alsKeys == nil {
		cache2go.RemPrefixKey(ACC_ALIAS_PREFIX)
	}
	if dcsKeys == nil {
		cache2go.RemPrefixKey(DERIVEDCHARGERS_PREFIX)
	}
	for k, _ := range ms.dict {
		if strings.HasPrefix(k, ACTION_PREFIX) {
			cache2go.RemKey(k)
			if _, err := ms.GetActions(k[len(ACTION_PREFIX):], true); err != nil {
				cache2go.RollbackTransaction()
				return err
			}
		}
		if strings.HasPrefix(k, SHARED_GROUP_PREFIX) {
			cache2go.RemKey(k)
			if _, err := ms.GetSharedGroup(k[len(SHARED_GROUP_PREFIX):], true); err != nil {
				cache2go.RollbackTransaction()
				return err
			}
		}
		if strings.HasPrefix(k, ACC_ALIAS_PREFIX) {
			cache2go.RemKey(k)
			if _, err := ms.GetAccAlias(k[len(ACC_ALIAS_PREFIX):], true); err != nil {
				cache2go.RollbackTransaction()
				return err
			}
		}
		if strings.HasPrefix(k, DERIVEDCHARGERS_PREFIX) {
			cache2go.RemKey(k)
			if _, err := ms.GetDerivedChargers(k[len(DERIVEDCHARGERS_PREFIX):], true); err != nil {
				cache2go.RollbackTransaction()
				return err
			}
		}
	}
	cache2go.CommitTransaction()
	return nil
}

// Used to check if specific subject is stored using prefix key attached to entity
func (ms *MapStorage) HasData(categ, subject string) (bool, error) {
	switch categ {
	case DESTINATION_PREFIX:
		_, exists := ms.dict[DESTINATION_PREFIX+subject]
		return exists, nil
	case RATING_PLAN_PREFIX:
		_, exists := ms.dict[RATING_PLAN_PREFIX+subject]
		return exists, nil
	}
	return false, errors.New("Unsupported category")
}

func (ms *MapStorage) GetRatingPlan(key string, skipCache bool) (rp *RatingPlan, err error) {
	key = RATING_PLAN_PREFIX + key
	if !skipCache {
		if x, err := cache2go.GetCached(key); err == nil {
			return x.(*RatingPlan), nil
		} else {
			return nil, err
		}
	}
	if values, ok := ms.dict[key]; ok {
		b := bytes.NewBuffer(values)
		r, err := zlib.NewReader(b)
		if err != nil {
			return nil, err
		}
		out, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		r.Close()
		rp = new(RatingPlan)
		err = ms.ms.Unmarshal(out, rp)
		cache2go.Cache(key, rp)
	} else {
		return nil, errors.New(utils.ERR_NOT_FOUND)
	}
	return
}

func (ms *MapStorage) SetRatingPlan(rp *RatingPlan, massPipe io.Writer) (err error) {
	result, err := ms.ms.Marshal(rp)
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(result)
	w.Close()
	ms.dict[RATING_PLAN_PREFIX+rp.Id] = b.Bytes()
	response := 0
	if historyScribe != nil {
		go historyScribe.Record(rp.GetHistoryRecord(), &response)
	}
	//cache2go.Cache(RATING_PLAN_PREFIX+rp.Id, rp)
	return
}

func (ms *MapStorage) GetRatingProfile(key string, skipCache bool) (rpf *RatingProfile, err error) {
	key = RATING_PROFILE_PREFIX + key
	if !skipCache {
		if x, err := cache2go.GetCached(key); err == nil {
			return x.(*RatingProfile), nil
		} else {
			return nil, err
		}
	}
	if values, ok := ms.dict[key]; ok {
		rpf = new(RatingProfile)

		err = ms.ms.Unmarshal(values, rpf)
		cache2go.Cache(key, rpf)
	} else {
		return nil, errors.New(utils.ERR_NOT_FOUND)
	}
	return
}

func (ms *MapStorage) SetRatingProfile(rpf *RatingProfile, massPipe io.Writer) (err error) {
	result, err := ms.ms.Marshal(rpf)
	ms.dict[RATING_PROFILE_PREFIX+rpf.Id] = result
	response := 0
	if historyScribe != nil {
		go historyScribe.Record(rpf.GetHistoryRecord(), &response)
	}
	//cache2go.Cache(RATING_PROFILE_PREFIX+rpf.Id, rpf)
	return
}

func (ms *MapStorage) GetLCR(key string, skipCache bool) (lcr *LCR, err error) {
	key = LCR_PREFIX + key
	if !skipCache {
		if x, err := cache2go.GetCached(key); err == nil {
			return x.(*LCR), nil
		} else {
			return nil, err
		}
	}
	if values, ok := ms.dict[key]; ok {
		err = ms.ms.Unmarshal(values, &lcr)
		cache2go.Cache(key, lcr)
	} else {
		return nil, errors.New(utils.ERR_NOT_FOUND)
	}
	return
}

func (ms *MapStorage) SetLCR(lcr *LCR, massPipe io.Writer) (err error) {
	result, err := ms.ms.Marshal(lcr)
	ms.dict[LCR_PREFIX+lcr.GetId()] = result
	//cache2go.Cache(LCR_PREFIX+key, lcr)
	return
}

func (ms *MapStorage) GetRpAlias(key string, skipCache bool) (alias string, err error) {
	key = RP_ALIAS_PREFIX + key
	if !skipCache {
		if x, err := cache2go.GetCached(key); err == nil {
			return x.(string), nil
		} else {
			return "", err
		}
	}
	if values, ok := ms.dict[key]; ok {
		alias = string(values)
		cache2go.Cache(key, alias)
	} else {
		return "", errors.New(utils.ERR_NOT_FOUND)
	}
	return
}

func (ms *MapStorage) SetRpAlias(key, alias string, massPipe io.Writer) (err error) {
	ms.dict[RP_ALIAS_PREFIX+key] = []byte(alias)
	//cache2go.Cache(ALIAS_PREFIX+key, alias)
	return
}

func (ms *MapStorage) RemoveRpAliases(tenantRtSubjects []*TenantRatingSubject) (err error) {
	for key, _ := range ms.dict {
		for _, tntRtSubj := range tenantRtSubjects {
			tenantPrfx := RP_ALIAS_PREFIX + tntRtSubj.Tenant + utils.CONCATENATED_KEY_SEP
			if strings.HasPrefix(key, RP_ALIAS_PREFIX) {
				alsSubj, err := ms.GetRpAlias(key[len(RP_ALIAS_PREFIX):], true)
				if err != nil {
					return err
				}
				if len(key) >= len(tenantPrfx) && key[:len(tenantPrfx)] == tenantPrfx && tntRtSubj.Subject == alsSubj {
					cache2go.RemKey(key)
					delete(ms.dict, key)
				}
			}
		}
	}
	return
}

func (ms *MapStorage) GetRPAliases(tenant, subject string, skipCache bool) (aliases []string, err error) {
	tenantPrfx := RP_ALIAS_PREFIX + tenant + utils.CONCATENATED_KEY_SEP
	var alsKeys []string
	if !skipCache {
		alsKeys = cache2go.GetEntriesKeys(tenantPrfx)
	}
	for _, key := range alsKeys {
		if alsSubj, err := ms.GetRpAlias(key[len(RP_ALIAS_PREFIX):], skipCache); err != nil {
			return nil, err
		} else if alsSubj == subject {
			alsFromKey := key[len(tenantPrfx):] // take out the alias out of key+tenant
			aliases = append(aliases, alsFromKey)
		}
	}
	if len(alsKeys) == 0 {
		for key, value := range ms.dict {
			if strings.HasPrefix(key, RP_ALIAS_PREFIX) && len(key) >= len(tenantPrfx) && key[:len(tenantPrfx)] == tenantPrfx && subject == string(value) {
				aliases = append(aliases, key[len(tenantPrfx):])
			}
		}
	}
	return aliases, nil
}

func (ms *MapStorage) GetAccAlias(key string, skipCache bool) (alias string, err error) {
	key = ACC_ALIAS_PREFIX + key
	if !skipCache {
		if x, err := cache2go.GetCached(key); err == nil {
			return x.(string), nil
		} else {
			return "", err
		}
	}
	if values, ok := ms.dict[key]; ok {
		alias = string(values)
		cache2go.Cache(key, alias)
	} else {
		return "", errors.New(utils.ERR_NOT_FOUND)
	}
	return
}

func (ms *MapStorage) SetAccAlias(key, alias string) (err error) {
	ms.dict[ACC_ALIAS_PREFIX+key] = []byte(alias)
	//cache2go.Cache(ALIAS_PREFIX+key, alias)
	return
}

func (ms *MapStorage) RemoveAccAliases(tenantAccounts []*TenantAccount) (err error) {
	for key, value := range ms.dict {
		for _, tntAcnt := range tenantAccounts {
			tenantPrfx := ACC_ALIAS_PREFIX + tntAcnt.Tenant + utils.CONCATENATED_KEY_SEP
			if strings.HasPrefix(key, ACC_ALIAS_PREFIX) && len(key) >= len(tenantPrfx) && key[:len(tenantPrfx)] == tenantPrfx && tntAcnt.Account == string(value) {
				delete(ms.dict, key)
			}
		}
	}
	return
}

func (ms *MapStorage) GetAccountAliases(tenant, account string, skipCache bool) (aliases []string, err error) {
	for key, value := range ms.dict {
		tenantPrfx := ACC_ALIAS_PREFIX + tenant + utils.CONCATENATED_KEY_SEP
		if strings.HasPrefix(key, ACC_ALIAS_PREFIX) && len(key) >= len(tenantPrfx) && key[:len(tenantPrfx)] == tenantPrfx && account == string(value) {
			aliases = append(aliases, key[len(tenantPrfx):])
		}
	}
	return aliases, nil
}

func (ms *MapStorage) GetDestination(key string) (dest *Destination, err error) {
	key = DESTINATION_PREFIX + key
	if values, ok := ms.dict[key]; ok {
		b := bytes.NewBuffer(values)
		r, err := zlib.NewReader(b)
		if err != nil {
			return nil, err
		}
		out, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		r.Close()
		dest = new(Destination)
		err = ms.ms.Unmarshal(out, dest)
		// create optimized structure
		for _, p := range dest.Prefixes {
			cache2go.CachePush(DESTINATION_PREFIX+p, dest.Id)
		}
	} else {
		return nil, errors.New(utils.ERR_NOT_FOUND)
	}
	return
}

func (ms *MapStorage) SetDestination(dest *Destination, massPipe io.Writer) (err error) {
	result, err := ms.ms.Marshal(dest)
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(result)
	w.Close()
	ms.dict[DESTINATION_PREFIX+dest.Id] = b.Bytes()
	response := 0
	if historyScribe != nil {
		go historyScribe.Record(dest.GetHistoryRecord(), &response)
	}
	//cache2go.Cache(DESTINATION_PREFIX+dest.Id, dest)
	return
}

func (ms *MapStorage) GetActions(key string, skipCache bool) (as Actions, err error) {
	key = ACTION_PREFIX + key
	if !skipCache {
		if x, err := cache2go.GetCached(key); err == nil {
			return x.(Actions), nil
		} else {
			return nil, err
		}
	}
	if values, ok := ms.dict[key]; ok {
		err = ms.ms.Unmarshal(values, &as)
		cache2go.Cache(key, as)
	} else {
		return nil, errors.New(utils.ERR_NOT_FOUND)
	}
	return
}

func (ms *MapStorage) SetActions(key string, as Actions) (err error) {
	result, err := ms.ms.Marshal(&as)
	ms.dict[ACTION_PREFIX+key] = result
	//cache2go.Cache(ACTION_PREFIX+key, as)
	return
}

func (ms *MapStorage) GetSharedGroup(key string, skipCache bool) (sg *SharedGroup, err error) {
	key = SHARED_GROUP_PREFIX + key
	if !skipCache {
		if x, err := cache2go.GetCached(key); err == nil {
			return x.(*SharedGroup), nil
		} else {
			return nil, err
		}
	}
	if values, ok := ms.dict[key]; ok {
		err = ms.ms.Unmarshal(values, &sg)
		cache2go.Cache(key, sg)
	} else {
		return nil, errors.New(utils.ERR_NOT_FOUND)
	}
	return
}

func (ms *MapStorage) SetSharedGroup(sg *SharedGroup) (err error) {
	result, err := ms.ms.Marshal(sg)
	ms.dict[SHARED_GROUP_PREFIX+sg.Id] = result
	//cache2go.Cache(SHARED_GROUP_PREFIX+key, sg)
	return
}

func (ms *MapStorage) GetAccount(key string) (ub *Account, err error) {
	if values, ok := ms.dict[ACCOUNT_PREFIX+key]; ok {
		ub = &Account{Id: key}
		err = ms.ms.Unmarshal(values, ub)
	} else {
		return nil, errors.New(utils.ERR_NOT_FOUND)
	}
	return
}

func (ms *MapStorage) SetAccount(ub *Account) (err error) {
	// never override existing account with an empty one
	// UPDATE: if all balances expired and were clean it makes
	// sense to write empty balance map
	if len(ub.BalanceMap) == 0 {
		if ac, err := ms.GetAccount(ub.Id); err == nil && !ac.allBalancesExpired() {
			ac.ActionTriggers = ub.ActionTriggers
			ac.UnitCounters = ub.UnitCounters
			ac.AllowNegative = ub.AllowNegative
			ac.Disabled = ub.Disabled
			ub = ac
		}
	}
	result, err := ms.ms.Marshal(ub)
	ms.dict[ACCOUNT_PREFIX+ub.Id] = result
	return
}

func (ms *MapStorage) GetActionTimings(key string) (ats ActionPlan, err error) {
	if values, ok := ms.dict[ACTION_TIMING_PREFIX+key]; ok {
		err = ms.ms.Unmarshal(values, &ats)
	} else {
		return nil, errors.New(utils.ERR_NOT_FOUND)
	}
	return
}

func (ms *MapStorage) SetActionTimings(key string, ats ActionPlan) (err error) {
	if len(ats) == 0 {
		// delete the key
		delete(ms.dict, ACTION_TIMING_PREFIX+key)
		return
	}
	result, err := ms.ms.Marshal(&ats)
	ms.dict[ACTION_TIMING_PREFIX+key] = result
	return
}

func (ms *MapStorage) GetAllActionTimings() (ats map[string]ActionPlan, err error) {
	ats = make(map[string]ActionPlan)
	for key, value := range ms.dict {
		if !strings.HasPrefix(key, ACTION_TIMING_PREFIX) {
			continue
		}
		var tempAts ActionPlan
		err = ms.ms.Unmarshal(value, &tempAts)
		ats[key[len(ACTION_TIMING_PREFIX):]] = tempAts
	}

	return
}

func (ms *MapStorage) GetDerivedChargers(key string, skipCache bool) (dcs utils.DerivedChargers, err error) {
	key = DERIVEDCHARGERS_PREFIX + key
	if !skipCache {
		if x, err := cache2go.GetCached(key); err == nil {
			return x.(utils.DerivedChargers), nil
		} else {
			return nil, err
		}
	}
	if values, ok := ms.dict[key]; ok {
		err = ms.ms.Unmarshal(values, &dcs)
		cache2go.Cache(key, dcs)
	} else {
		return nil, errors.New(utils.ERR_NOT_FOUND)
	}
	return
}

func (ms *MapStorage) SetDerivedChargers(key string, dcs utils.DerivedChargers) error {
	result, err := ms.ms.Marshal(dcs)
	ms.dict[DERIVEDCHARGERS_PREFIX+key] = result
	return err
}

func (ms *MapStorage) SetCdrStats(cs *CdrStats, massPipe io.Writer) error {
	result, err := ms.ms.Marshal(cs)
	ms.dict[CDR_STATS_PREFIX+cs.Id] = result
	return err
}

func (ms *MapStorage) GetCdrStats(key string) (cs *CdrStats, err error) {
	if values, ok := ms.dict[CDR_STATS_PREFIX+key]; ok {
		err = ms.ms.Unmarshal(values, &cs)
	} else {
		return nil, errors.New(utils.ERR_NOT_FOUND)
	}
	return
}

func (ms *MapStorage) GetAllCdrStats() (css []*CdrStats, err error) {
	for key, value := range ms.dict {
		if !strings.HasPrefix(key, CDR_STATS_PREFIX) {
			continue
		}
		cs := &CdrStats{}
		err = ms.ms.Unmarshal(value, cs)
		css = append(css, cs)
	}
	return
}

func (ms *MapStorage) LogCallCost(cgrid, source, runid string, cc *CallCost) error {
	result, err := ms.ms.Marshal(cc)
	ms.dict[LOG_CALL_COST_PREFIX+source+runid+"_"+cgrid] = result
	return err
}

func (ms *MapStorage) GetCallCostLog(cgrid, source, runid string) (cc *CallCost, err error) {
	if values, ok := ms.dict[LOG_CALL_COST_PREFIX+source+runid+"_"+cgrid]; ok {
		err = ms.ms.Unmarshal(values, &cc)
	} else {
		return nil, errors.New(utils.ERR_NOT_FOUND)
	}
	return
}

func (ms *MapStorage) LogActionTrigger(ubId, source string, at *ActionTrigger, as Actions) (err error) {
	mat, err := ms.ms.Marshal(at)
	if err != nil {
		return
	}
	mas, err := ms.ms.Marshal(&as)
	if err != nil {
		return
	}
	ms.dict[LOG_ACTION_TRIGGER_PREFIX+source+"_"+time.Now().Format(time.RFC3339Nano)] = []byte(fmt.Sprintf("%s*%s*%s", ubId, string(mat), string(mas)))
	return
}

func (ms *MapStorage) LogActionTiming(source string, at *ActionTiming, as Actions) (err error) {
	mat, err := ms.ms.Marshal(at)
	if err != nil {
		return
	}
	mas, err := ms.ms.Marshal(&as)
	if err != nil {
		return
	}
	ms.dict[LOG_ACTION_TIMMING_PREFIX+source+"_"+time.Now().Format(time.RFC3339Nano)] = []byte(fmt.Sprintf("%s*%s", string(mat), string(mas)))
	return
}

func (ms *MapStorage) LogError(uuid, source, runid, errstr string) (err error) {
	ms.dict[LOG_ERR+source+runid+"_"+uuid] = []byte(errstr)
	return nil
}

func (ms *MapStorage) RatingMassInsert(loader TPLoader, flush, verbose bool) error {
	loader.WriteRating(nil, flush, verbose)
	return nil
}

func (ms *MapStorage) AccountingMassInsert(loader TPLoader, verbose bool) error {
	loader.WriteAccounting(nil, verbose)
	return nil
}
