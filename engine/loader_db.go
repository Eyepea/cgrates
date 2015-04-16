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
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/cgrates/cgrates/utils"
)

type DbReader struct {
	tpid             string
	storDb           LoadStorage
	dataDb           RatingStorage
	accountDb        AccountingStorage
	actions          map[string][]*Action
	actionsTimings   map[string][]*ActionTiming
	actionsTriggers  map[string][]*ActionTrigger
	accountActions   map[string]*Account
	dirtyRpAliases   []*TenantRatingSubject // used to clean aliases that might have changed
	dirtyAccAliases  []*TenantAccount       // used to clean aliases that might have changed
	destinations     map[string]*Destination
	rpAliases        map[string]string
	accAliases       map[string]string
	timings          map[string]*utils.TPTiming
	rates            map[string]*utils.TPRate
	destinationRates map[string]*utils.TPDestinationRate
	ratingPlans      map[string]*RatingPlan
	ratingProfiles   map[string]*RatingProfile
	sharedGroups     map[string]*SharedGroup
	lcrs             map[string]*LCR
	derivedChargers  map[string]utils.DerivedChargers
	cdrStats         map[string]*CdrStats
}

func NewDbReader(storDB LoadStorage, ratingDb RatingStorage, accountDb AccountingStorage, tpid string) *DbReader {
	c := new(DbReader)
	c.storDb = storDB
	c.dataDb = ratingDb
	c.accountDb = accountDb
	c.tpid = tpid
	c.actions = make(map[string][]*Action)
	c.actionsTimings = make(map[string][]*ActionTiming)
	c.actionsTriggers = make(map[string][]*ActionTrigger)
	c.ratingPlans = make(map[string]*RatingPlan)
	c.ratingProfiles = make(map[string]*RatingProfile)
	c.sharedGroups = make(map[string]*SharedGroup)
	c.lcrs = make(map[string]*LCR)
	c.rpAliases = make(map[string]string)
	c.accAliases = make(map[string]string)
	c.timings = make(map[string]*utils.TPTiming)
	c.accountActions = make(map[string]*Account)
	c.destinations = make(map[string]*Destination)
	c.cdrStats = make(map[string]*CdrStats)
	c.derivedChargers = make(map[string]utils.DerivedChargers)
	return c
}

// FIXME: this method is code duplication from csv loader
func (dbr *DbReader) ShowStatistics() {
	// destinations
	destCount := len(dbr.destinations)
	log.Print("Destinations: ", destCount)
	prefixDist := make(map[int]int, 50)
	prefixCount := 0
	for _, d := range dbr.destinations {
		prefixDist[len(d.Prefixes)] += 1
		prefixCount += len(d.Prefixes)
	}
	log.Print("Avg Prefixes: ", prefixCount/destCount)
	log.Print("Prefixes distribution:")
	for k, v := range prefixDist {
		log.Printf("%d: %d", k, v)
	}
	// rating plans
	rplCount := len(dbr.ratingPlans)
	log.Print("Rating plans: ", rplCount)
	destRatesDist := make(map[int]int, 50)
	destRatesCount := 0
	for _, rpl := range dbr.ratingPlans {
		destRatesDist[len(rpl.DestinationRates)] += 1
		destRatesCount += len(rpl.DestinationRates)
	}
	log.Print("Avg Destination Rates: ", destRatesCount/rplCount)
	log.Print("Destination Rates distribution:")
	for k, v := range destRatesDist {
		log.Printf("%d: %d", k, v)
	}
	// rating profiles
	rpfCount := len(dbr.ratingProfiles)
	log.Print("Rating profiles: ", rpfCount)
	activDist := make(map[int]int, 50)
	activCount := 0
	for _, rpf := range dbr.ratingProfiles {
		activDist[len(rpf.RatingPlanActivations)] += 1
		activCount += len(rpf.RatingPlanActivations)
	}
	log.Print("Avg Activations: ", activCount/rpfCount)
	log.Print("Activation distribution:")
	for k, v := range activDist {
		log.Printf("%d: %d", k, v)
	}
	// actions
	log.Print("Actions: ", len(dbr.actions))
	// action timings
	log.Print("Action plans: ", len(dbr.actionsTimings))
	// account actions
	log.Print("Account actions: ", len(dbr.accountActions))
	// derivedChargers
	log.Print("Derived Chargers: ", len(dbr.derivedChargers))
	// lcr rules
	log.Print("LCR rules: ", len(dbr.lcrs))
}
func (dbr *DbReader) WriteToDatabase(flush, verbose bool) (err error) {
	if err := dbr.dataDb.RatingMassInsert(dbr, flush, verbose); err != nil {
		return err
	}
	if err := dbr.accountDb.AccountingMassInsert(dbr, verbose); err != nil {
		return err
	}
	return nil
}

func (dbr *DbReader) WriteAccounting(massPipe io.Writer, verbose bool) (err error) {
	accountDb := dbr.accountDb
	if verbose {
		log.Print("Action Plans:")
	}
	for k, ats := range dbr.actionsTimings {
		err = accountDb.SetActionTimings(k, ats)
		if err != nil {
			return err
		}
		if verbose {
			log.Println("\t", k)
		}
	}
	if verbose {
		log.Print("Shared Groups:")
	}
	for k, sg := range dbr.sharedGroups {
		err = accountDb.SetSharedGroup(sg)
		if err != nil {
			return err
		}
		if verbose {
			log.Println("\t", k)
		}
	}
	if verbose {
		log.Print("Actions:")
	}
	for k, as := range dbr.actions {
		err = accountDb.SetActions(k, as)
		if err != nil {
			return err
		}
		if verbose {
			log.Println("\t", k)
		}
	}
	if verbose {
		log.Print("Account Actions:")
	}
	for _, ub := range dbr.accountActions {
		err = accountDb.SetAccount(ub)
		if err != nil {
			return err
		}
		if verbose {
			log.Println("\t", ub.Id)
		}
	}

	if verbose {
		log.Print("Account Aliases:")
	}
	if err := accountDb.RemoveAccAliases(dbr.dirtyAccAliases); err != nil {
		return err
	}
	for key, alias := range dbr.accAliases {
		err = accountDb.SetAccAlias(key, alias)
		if err != nil {
			return err
		}
		if verbose {
			log.Print("\t", key)
		}
	}
	if verbose {
		log.Print("Derived Chargers:")
	}
	for key, dcs := range dbr.derivedChargers {
		err = accountDb.SetDerivedChargers(key, dcs)
		if err != nil {
			return err
		}
		if verbose {
			log.Print("\t", key)
		}
	}

	return
}

func (dbr *DbReader) WriteRating(massPipe io.Writer, flush, verbose bool) (err error) {
	dataDb := dbr.dataDb
	if dataDb == nil {
		return errors.New("No database connection!")
	}
	if flush {
		dataDb.Flush("")
	}
	if verbose {
		log.Print("Destinations:")
	}
	for _, d := range dbr.destinations {
		err = dataDb.SetDestination(d, massPipe)
		if err != nil {
			return err
		}
		if verbose {
			log.Print("\t", d.Id, " : ", d.Prefixes)
		}
	}
	if verbose {
		log.Print("Rating Plans:")
	}
	for _, rp := range dbr.ratingPlans {
		err = dataDb.SetRatingPlan(rp, massPipe)
		if err != nil {
			return err
		}
		if verbose {
			log.Print("\t", rp.Id)
		}
	}
	if verbose {
		log.Print("Rating Profiles:")
	}
	for _, rp := range dbr.ratingProfiles {
		err = dataDb.SetRatingProfile(rp, massPipe)
		if err != nil {
			return err
		}
		if verbose {
			log.Print("\t", rp.Id)
		}
	}
	if verbose {
		log.Print("Rating Profile Aliases:")
	}
	if err := dataDb.RemoveRpAliases(dbr.dirtyRpAliases); err != nil {
		return err
	}
	for key, alias := range dbr.rpAliases {
		err = dataDb.SetRpAlias(key, alias, massPipe)
		if err != nil {
			return err
		}
		if verbose {
			log.Print("\t", key)
		}
	}
	if verbose {
		log.Print("LCR Rules:")
	}
	for k, lcr := range dbr.lcrs {
		err = dataDb.SetLCR(lcr, massPipe)
		if err != nil {
			return err
		}
		if verbose {
			log.Println("\t", k)
		}
	}
	if verbose {
		log.Print("CDR Stats Queues:")
	}
	for _, sq := range dbr.cdrStats {
		err = dataDb.SetCdrStats(sq, massPipe)
		if err != nil {
			return err
		}
		if verbose {
			log.Print("\t", sq.Id)
		}
	}

	return nil
}

func (dbr *DbReader) LoadDestinations() (err error) {
	dbr.destinations, err = dbr.storDb.GetTpDestinations(dbr.tpid, "")
	return
}

func (dbr *DbReader) LoadDestinationByTag(tag string) (bool, error) {
	destinations, err := dbr.storDb.GetTpDestinations(dbr.tpid, tag)
	for _, destination := range destinations {
		dbr.dataDb.SetDestination(destination, nil)
	}
	return len(destinations) > 0, err
}

func (dbr *DbReader) LoadTimings() (err error) {
	tpTmgs, err := dbr.storDb.GetTpTimings(dbr.tpid, "")
	if err != nil {
		return err
	}
	for _, tpTm := range tpTmgs {
		dbr.timings[tpTm.TimingId] = NewTiming(tpTm.TimingId, tpTm.Years, tpTm.Months, tpTm.MonthDays, tpTm.WeekDays, tpTm.Time)
	}
	return nil
}

func (dbr *DbReader) LoadRates() (err error) {
	dbr.rates, err = dbr.storDb.GetTpRates(dbr.tpid, "")
	return err
}

func (dbr *DbReader) LoadDestinationRates() (err error) {
	dbr.destinationRates, err = dbr.storDb.GetTpDestinationRates(dbr.tpid, "", nil)
	if err != nil {
		return err
	}
	for _, drs := range dbr.destinationRates {
		for _, dr := range drs.DestinationRates {
			rate, exists := dbr.rates[dr.RateId]
			if !exists {
				return fmt.Errorf("Could not find rate for tag %v", dr.RateId)
			}
			dr.Rate = rate
			destinationExists := dr.DestinationId == utils.ANY
			if !destinationExists {
				_, destinationExists = dbr.destinations[dr.DestinationId]
			}
			if !destinationExists {
				if dbExists, err := dbr.dataDb.HasData(DESTINATION_PREFIX, dr.DestinationId); err != nil {
					return err
				} else if !dbExists {
					return fmt.Errorf("Could not get destination for tag %v", dr.DestinationId)
				}
			}
		}
	}
	return nil
}

func (dbr *DbReader) LoadRatingPlans() error {
	mpRpls, err := dbr.storDb.GetTpRatingPlans(dbr.tpid, "", nil)
	if err != nil {
		return err
	}
	for tag, rplBnds := range mpRpls {
		for _, rplBnd := range rplBnds {
			t, exists := dbr.timings[rplBnd.TimingId]
			if !exists {
				return fmt.Errorf("Could not get timing for tag %v", rplBnd.TimingId)
			}
			rplBnd.SetTiming(t)
			drs, exists := dbr.destinationRates[rplBnd.DestinationRatesId]
			if !exists {
				return fmt.Errorf("Could not find destination rate for tag %v", rplBnd.DestinationRatesId)
			}
			plan, exists := dbr.ratingPlans[tag]
			if !exists {
				plan = &RatingPlan{Id: tag}
				dbr.ratingPlans[plan.Id] = plan
			}
			for _, dr := range drs.DestinationRates {
				plan.AddRateInterval(dr.DestinationId, GetRateInterval(rplBnd, dr))
			}
		}
	}
	return nil
}

func (dbr *DbReader) LoadRatingProfiles() error {
	mpTpRpfs, err := dbr.storDb.GetTpRatingProfiles(&utils.TPRatingProfile{TPid: dbr.tpid}) //map[string]*utils.TPRatingProfile
	if err != nil {
		return err
	}
	for _, tpRpf := range mpTpRpfs {
		// extract aliases from subject
		aliases := strings.Split(tpRpf.Subject, ";")
		dbr.dirtyRpAliases = append(dbr.dirtyRpAliases, &TenantRatingSubject{Tenant: tpRpf.Tenant, Subject: aliases[0]})
		if len(aliases) > 1 {
			tpRpf.Subject = aliases[0]
			for _, alias := range aliases[1:] {
				dbr.rpAliases[utils.RatingSubjectAliasKey(tpRpf.Tenant, alias)] = tpRpf.Subject
			}
		}
		rpf := &RatingProfile{Id: tpRpf.KeyId()}
		for _, tpRa := range tpRpf.RatingPlanActivations {
			at, err := utils.ParseDate(tpRa.ActivationTime)
			if err != nil {
				return fmt.Errorf("Cannot parse activation time from %v", tpRa.ActivationTime)
			}
			_, exists := dbr.ratingPlans[tpRa.RatingPlanId]
			if !exists {
				if dbExists, err := dbr.dataDb.HasData(RATING_PLAN_PREFIX, tpRa.RatingPlanId); err != nil {
					return err
				} else if !dbExists {
					return fmt.Errorf("Could not load rating plans for tag: %v", tpRa.RatingPlanId)
				}
			}
			rpf.RatingPlanActivations = append(rpf.RatingPlanActivations,
				&RatingPlanActivation{
					ActivationTime:  at,
					RatingPlanId:    tpRa.RatingPlanId,
					FallbackKeys:    utils.FallbackSubjKeys(tpRpf.Direction, tpRpf.Tenant, tpRpf.Category, tpRa.FallbackSubjects),
					CdrStatQueueIds: strings.Split(tpRa.CdrStatQueueIds, utils.INFIELD_SEP),
				})
		}
		dbr.ratingProfiles[tpRpf.KeyId()] = rpf
	}
	return nil
}

// Returns true, nil in case of load success, false, nil in case of RatingPlan  not found in storDb
func (dbr *DbReader) LoadRatingPlanByTag(tag string) (bool, error) {
	mpRpls, err := dbr.storDb.GetTpRatingPlans(dbr.tpid, tag, nil)
	if err != nil {
		return false, err
	} else if len(mpRpls) == 0 {
		return false, nil
	}
	for tag, rplBnds := range mpRpls {
		ratingPlan := &RatingPlan{Id: tag}
		for _, rp := range rplBnds {
			tm, err := dbr.storDb.GetTpTimings(dbr.tpid, rp.TimingId)
			if err != nil || len(tm) == 0 {
				return false, fmt.Errorf("No Timings profile with id %s: %v", rp.TimingId, err)
			}
			tpTmng := NewTiming(tm[rp.TimingId].TimingId, tm[rp.TimingId].Years, tm[rp.TimingId].Months, tm[rp.TimingId].MonthDays, tm[rp.TimingId].WeekDays, tm[rp.TimingId].Time)
			rp.SetTiming(tpTmng)
			drm, err := dbr.storDb.GetTpDestinationRates(dbr.tpid, rp.DestinationRatesId, nil)
			if err != nil || len(drm) == 0 {
				return false, fmt.Errorf("No DestinationRates profile with id %s: %v", rp.DestinationRatesId, err)
			}
			for _, drate := range drm[rp.DestinationRatesId].DestinationRates {
				rt, err := dbr.storDb.GetTpRates(dbr.tpid, drate.RateId)
				if err != nil || len(rt) == 0 {
					return false, fmt.Errorf("No Rates profile with id %s: %v", drate.RateId, err)
				}
				drate.Rate = rt[drate.RateId]
				ratingPlan.AddRateInterval(drate.DestinationId, GetRateInterval(rp, drate))
				if drate.DestinationId == utils.ANY {
					continue // no need of loading the destinations in this case
				}
				dms, err := dbr.storDb.GetTpDestinations(dbr.tpid, drate.DestinationId)
				if err != nil {
					return false, err
				} else if len(dms) == 0 {
					if dbExists, err := dbr.dataDb.HasData(DESTINATION_PREFIX, drate.DestinationId); err != nil {
						return false, err
					} else if !dbExists {
						return false, fmt.Errorf("Could not get destination for tag %v", drate.DestinationId)
					}
					continue
				}
				for _, destination := range dms {
					dbr.dataDb.SetDestination(destination, nil)
				}
			}
		}
		if err := dbr.dataDb.SetRatingPlan(ratingPlan, nil); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (dbr *DbReader) LoadRatingProfileFiltered(qriedRpf *utils.TPRatingProfile) error {
	var resultRatingProfile *RatingProfile
	mpTpRpfs, err := dbr.storDb.GetTpRatingProfiles(qriedRpf) //map[string]*utils.TPRatingProfile
	if err != nil {
		return fmt.Errorf("No RateProfile for filter %v, error: %v", qriedRpf, err)
	}
	for _, tpRpf := range mpTpRpfs {
		resultRatingProfile = &RatingProfile{Id: tpRpf.KeyId()}
		for _, tpRa := range tpRpf.RatingPlanActivations {
			at, err := utils.ParseDate(tpRa.ActivationTime)
			if err != nil {
				return fmt.Errorf("Cannot parse activation time from %v", tpRa.ActivationTime)
			}
			_, exists := dbr.ratingPlans[tpRa.RatingPlanId]
			if !exists {
				if dbExists, err := dbr.dataDb.HasData(RATING_PLAN_PREFIX, tpRa.RatingPlanId); err != nil {
					return err
				} else if !dbExists {
					return fmt.Errorf("Could not load rating plans for tag: %v", tpRa.RatingPlanId)
				}
			}
			resultRatingProfile.RatingPlanActivations = append(resultRatingProfile.RatingPlanActivations,
				&RatingPlanActivation{
					ActivationTime:  at,
					RatingPlanId:    tpRa.RatingPlanId,
					FallbackKeys:    utils.FallbackSubjKeys(tpRpf.Direction, tpRpf.Tenant, tpRpf.Category, tpRa.FallbackSubjects),
					CdrStatQueueIds: strings.Split(tpRa.CdrStatQueueIds, utils.INFIELD_SEP),
				})
		}
		if err := dbr.dataDb.SetRatingProfile(resultRatingProfile, nil); err != nil {
			return err
		}
	}
	return nil
}

func (dbr *DbReader) LoadSharedGroupByTag(tag string, save bool) error {
	storSgs, err := dbr.storDb.GetTpSharedGroups(dbr.tpid, tag)
	if err != nil {
		return err
	}
	var loadedTags []string
	for tag, tpSgs := range storSgs {
		sg, exists := dbr.sharedGroups[tag]
		if !exists {
			sg = &SharedGroup{
				Id:                tag,
				AccountParameters: make(map[string]*SharingParameters, len(tpSgs)),
			}
		}
		for _, tpSg := range tpSgs {
			sg.AccountParameters[tpSg.Account] = &SharingParameters{
				Strategy:      tpSg.Strategy,
				RatingSubject: tpSg.RatingSubject,
			}
		}
		dbr.sharedGroups[tag] = sg
		loadedTags = append(loadedTags, tag)
	}
	if save {
		for _, tag := range loadedTags {
			if err := dbr.accountDb.SetSharedGroup(dbr.sharedGroups[tag]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (dbr *DbReader) LoadSharedGroups() error {
	return dbr.LoadSharedGroupByTag("", false)
}

func (dbr *DbReader) LoadLCRs() (err error) {
	dbr.lcrs, err = dbr.storDb.GetTpLCRs(dbr.tpid, "")
	return err
}

func (dbr *DbReader) LoadActions() (err error) {
	storActs, err := dbr.storDb.GetTpActions(dbr.tpid, "")
	if err != nil {
		return err
	}
	// map[string][]*Action
	for tag, tpacts := range storActs {
		acts := make([]*Action, len(tpacts))
		for idx, tpact := range tpacts {
			acts[idx] = &Action{
				Id:               tag + strconv.Itoa(idx),
				ActionType:       tpact.Identifier,
				BalanceType:      tpact.BalanceType,
				Direction:        tpact.Direction,
				Weight:           tpact.Weight,
				ExtraParameters:  tpact.ExtraParameters,
				ExpirationString: tpact.ExpiryTime,
				Balance: &Balance{
					Uuid:          utils.GenUUID(),
					Id:            tpact.BalanceId,
					Value:         tpact.Units,
					Weight:        tpact.BalanceWeight,
					TimingIDs:     tpact.TimingTags,
					RatingSubject: tpact.RatingSubject,
					Category:      tpact.Category,
					DestinationId: tpact.DestinationId,
				},
			}
			// load action timings from tags
			if acts[idx].Balance.TimingIDs != "" {
				timingIds := strings.Split(acts[idx].Balance.TimingIDs, utils.INFIELD_SEP)
				for _, timingID := range timingIds {
					if timing, found := dbr.timings[timingID]; found {
						acts[idx].Balance.Timings = append(acts[idx].Balance.Timings, &RITiming{
							Years:     timing.Years,
							Months:    timing.Months,
							MonthDays: timing.MonthDays,
							WeekDays:  timing.WeekDays,
							StartTime: timing.StartTime,
							EndTime:   timing.EndTime,
						})
					} else {
						return fmt.Errorf("Could not find timing: %v", timingID)
					}
				}
			}
		}
		dbr.actions[tag] = acts
	}
	return nil
}

func (dbr *DbReader) LoadActionTimings() (err error) {
	atsMap, err := dbr.storDb.GetTPActionTimings(dbr.tpid, "")
	if err != nil {
		return err
	}
	for atId, ats := range atsMap {
		for _, at := range ats {

			_, exists := dbr.actions[at.ActionsId]
			if !exists {
				return fmt.Errorf("ActionTiming: Could not load the action for tag: %v", at.ActionsId)
			}
			t, exists := dbr.timings[at.TimingId]
			if !exists {
				return fmt.Errorf("ActionTiming: Could not load the timing for tag: %v", at.TimingId)
			}
			actTmg := &ActionTiming{
				Uuid:   utils.GenUUID(),
				Id:     atId,
				Weight: at.Weight,
				Timing: &RateInterval{
					Timing: &RITiming{
						Years:     t.Years,
						Months:    t.Months,
						MonthDays: t.MonthDays,
						WeekDays:  t.WeekDays,
						StartTime: t.StartTime,
					},
				},
				ActionsId: at.ActionsId,
			}
			dbr.actionsTimings[atId] = append(dbr.actionsTimings[atId], actTmg)
		}
	}
	return err
}

func (dbr *DbReader) LoadActionTriggers() (err error) {
	atrsMap, err := dbr.storDb.GetTpActionTriggers(dbr.tpid, "")
	if err != nil {
		return err
	}
	for key, atrsLst := range atrsMap {
		atrs := make([]*ActionTrigger, len(atrsLst))
		for idx, apiAtr := range atrsLst {
			balance_expiration_date, _ := utils.ParseTimeDetectLayout(apiAtr.BalanceExpirationDate)
			id := apiAtr.Id
			if id == "" {
				id = utils.GenUUID()
			}
			minSleep, err := utils.ParseDurationWithSecs(apiAtr.MinSleep)
			if err != nil {
				return err
			}
			atrs[idx] = &ActionTrigger{
				Id:                    id,
				ThresholdType:         apiAtr.ThresholdType,
				ThresholdValue:        apiAtr.ThresholdValue,
				Recurrent:             apiAtr.Recurrent,
				MinSleep:              minSleep,
				BalanceId:             apiAtr.BalanceId,
				BalanceType:           apiAtr.BalanceType,
				BalanceDirection:      apiAtr.BalanceDirection,
				BalanceDestinationId:  apiAtr.BalanceDestinationId,
				BalanceWeight:         apiAtr.BalanceWeight,
				BalanceExpirationDate: balance_expiration_date,
				BalanceTimingTags:     apiAtr.BalanceTimingTags,
				BalanceRatingSubject:  apiAtr.BalanceRatingSubject,
				BalanceCategory:       apiAtr.BalanceCategory,
				BalanceSharedGroup:    apiAtr.BalanceSharedGroup,
				Weight:                apiAtr.Weight,
				ActionsId:             apiAtr.ActionsId,
				MinQueuedItems:        apiAtr.MinQueuedItems,
			}
			if atrs[idx].Id == "" {
				atrs[idx].Id = utils.GenUUID()
			}
		}
		dbr.actionsTriggers[key] = atrs
	}
	return err
}

func (dbr *DbReader) LoadAccountActions() (err error) {
	acs, err := dbr.storDb.GetTpAccountActions(&utils.TPAccountActions{TPid: dbr.tpid})
	if err != nil {
		return err
	}
	for _, aa := range acs {
		if _, alreadyDefined := dbr.accountActions[aa.KeyId()]; alreadyDefined {
			return fmt.Errorf("Duplicate account action found: %s", aa.KeyId())
		}

		// extract aliases from subject
		aliases := strings.Split(aa.Account, ";")
		dbr.dirtyAccAliases = append(dbr.dirtyAccAliases, &TenantAccount{Tenant: aa.Tenant, Account: aliases[0]})
		if len(aliases) > 1 {
			aa.Account = aliases[0]
			for _, alias := range aliases[1:] {
				dbr.accAliases[utils.AccountAliasKey(aa.Tenant, alias)] = aa.Account
			}
		}
		aTriggers, exists := dbr.actionsTriggers[aa.ActionTriggersId]
		if !exists {
			return fmt.Errorf("Could not get action triggers for tag %v", aa.ActionTriggersId)
		}
		ub := &Account{
			Id:             aa.KeyId(),
			ActionTriggers: aTriggers,
		}
		dbr.accountActions[aa.KeyId()] = ub
		aTimings, exists := dbr.actionsTimings[aa.ActionPlanId]
		if !exists {
			log.Printf("Could not get action timing for tag %v", aa.ActionPlanId)
			// must not continue here
		}
		for _, at := range aTimings {
			at.AccountIds = append(at.AccountIds, aa.KeyId())
		}
	}
	return nil
}

func (dbr *DbReader) LoadAccountActionsFiltered(qriedAA *utils.TPAccountActions) error {
	accountActions, err := dbr.storDb.GetTpAccountActions(qriedAA)
	if err != nil {
		return errors.New(err.Error() + ": " + fmt.Sprintf("%+v", qriedAA))
	}
	for _, accountAction := range accountActions {
		id := accountAction.KeyId()
		var actionsIds []string // collects action ids
		// action timings
		if accountAction.ActionPlanId != "" {
			// get old userBalanceIds
			var exitingAccountIds []string
			existingActionTimings, err := dbr.accountDb.GetActionTimings(accountAction.ActionPlanId)
			if err == nil && len(existingActionTimings) > 0 {
				// all action timings from a specific tag shuld have the same list of user balances from the first one
				exitingAccountIds = existingActionTimings[0].AccountIds
			}

			actionTimingsMap, err := dbr.storDb.GetTPActionTimings(dbr.tpid, accountAction.ActionPlanId)
			if err != nil {
				return errors.New(err.Error() + " (ActionPlan): " + accountAction.ActionPlanId)
			} else if len(actionTimingsMap) == 0 {
				return fmt.Errorf("No ActionTimings with id <%s>", accountAction.ActionPlanId)
			}
			var actionTimings []*ActionTiming
			ats := actionTimingsMap[accountAction.ActionPlanId]
			for _, at := range ats {
				// Check action exists before saving it inside actionTiming key
				// ToDo: try saving the key after the actions was retrieved in order to save one query here.
				if actions, err := dbr.storDb.GetTpActions(dbr.tpid, at.ActionsId); err != nil {
					return errors.New(err.Error() + " (Actions): " + at.ActionsId)
				} else if len(actions) == 0 {
					return fmt.Errorf("No Action with id <%s>", at.ActionsId)
				}
				timingsMap, err := dbr.storDb.GetTpTimings(dbr.tpid, at.TimingId)
				if err != nil {
					return errors.New(err.Error() + " (Timing): " + at.TimingId)
				} else if len(timingsMap) == 0 {
					return fmt.Errorf("No Timing with id <%s>", at.TimingId)
				}
				t := NewTiming(timingsMap[at.TimingId].TimingId, timingsMap[at.TimingId].Years, timingsMap[at.TimingId].Months, timingsMap[at.TimingId].MonthDays, timingsMap[at.TimingId].WeekDays, timingsMap[at.TimingId].Time)
				actTmg := &ActionTiming{
					Uuid:   utils.GenUUID(),
					Id:     accountAction.ActionPlanId,
					Weight: at.Weight,
					Timing: &RateInterval{
						Timing: &RITiming{
							Months:    t.Months,
							MonthDays: t.MonthDays,
							WeekDays:  t.WeekDays,
							StartTime: t.StartTime,
						},
					},
					ActionsId: at.ActionsId,
				}
				// collect action ids from timings
				actionsIds = append(actionsIds, actTmg.ActionsId)
				//add user balance id if no already in
				found := false
				for _, ubId := range exitingAccountIds {
					if ubId == id {
						found = true
						break
					}
				}
				if !found {
					actTmg.AccountIds = append(exitingAccountIds, id)
				}
				actionTimings = append(actionTimings, actTmg)
			}

			// write action timings
			err = dbr.accountDb.SetActionTimings(accountAction.ActionPlanId, actionTimings)
			if err != nil {
				return errors.New(err.Error() + " (SetActionPlan): " + accountAction.ActionPlanId)
			}
		}
		// action triggers
		var actionTriggers ActionTriggerPriotityList
		//ActionTriggerPriotityList []*ActionTrigger
		if accountAction.ActionTriggersId != "" {
			apiAtrsMap, err := dbr.storDb.GetTpActionTriggers(dbr.tpid, accountAction.ActionTriggersId)
			if err != nil {
				return errors.New(err.Error() + " (ActionTriggers): " + accountAction.ActionTriggersId)
			}
			atrsMap := make(map[string][]*ActionTrigger)
			for key, atrsLst := range apiAtrsMap {
				atrs := make([]*ActionTrigger, len(atrsLst))
				for idx, apiAtr := range atrsLst {
					expTime, _ := utils.ParseDate(apiAtr.BalanceExpirationDate)
					atrs[idx] = &ActionTrigger{Id: utils.GenUUID(),
						ThresholdType:         apiAtr.ThresholdType,
						ThresholdValue:        apiAtr.ThresholdValue,
						BalanceId:             apiAtr.BalanceId,
						BalanceType:           apiAtr.BalanceType,
						BalanceDirection:      apiAtr.BalanceDirection,
						BalanceDestinationId:  apiAtr.BalanceDestinationId,
						BalanceWeight:         apiAtr.BalanceWeight,
						BalanceExpirationDate: expTime,
						BalanceRatingSubject:  apiAtr.BalanceRatingSubject,
						BalanceCategory:       apiAtr.BalanceCategory,
						BalanceSharedGroup:    apiAtr.BalanceSharedGroup,
						Weight:                apiAtr.Weight,
						ActionsId:             apiAtr.ActionsId,
					}
				}
				atrsMap[key] = atrs
			}
			actionTriggers = atrsMap[accountAction.ActionTriggersId]
			// collect action ids from triggers
			for _, atr := range actionTriggers {
				actionsIds = append(actionsIds, atr.ActionsId)
			}
		}

		// actions
		acts := make(map[string][]*Action)
		for _, actId := range actionsIds {
			storActs, err := dbr.storDb.GetTpActions(dbr.tpid, actId)
			if err != nil {
				return err
			}
			for tag, tpacts := range storActs {
				enacts := make([]*Action, len(tpacts))
				for idx, tpact := range tpacts {
					enacts[idx] = &Action{
						Id:               tag + strconv.Itoa(idx),
						ActionType:       tpact.Identifier,
						BalanceType:      tpact.BalanceType,
						Direction:        tpact.Direction,
						Weight:           tpact.Weight,
						ExtraParameters:  tpact.ExtraParameters,
						ExpirationString: tpact.ExpiryTime,
						Balance: &Balance{
							Uuid:          utils.GenUUID(),
							Value:         tpact.Units,
							Weight:        tpact.BalanceWeight,
							RatingSubject: tpact.RatingSubject,
							DestinationId: tpact.DestinationId,
						},
					}
				}
				acts[tag] = enacts
			}
		}
		// writee actions
		for k, as := range acts {
			err = dbr.accountDb.SetActions(k, as)
			if err != nil {
				return err
			}
		}
		ub, err := dbr.accountDb.GetAccount(id)
		if err != nil {
			ub = &Account{
				Id: id,
			}
		}
		ub.ActionTriggers = actionTriggers

		if err := dbr.accountDb.SetAccount(ub); err != nil {
			return err
		}
	}
	return nil
}

func (dbr *DbReader) LoadDerivedChargers() (err error) {
	return dbr.LoadDerivedChargersFiltered(&utils.TPDerivedChargers{TPid: dbr.tpid}, false)
}

func (dbr *DbReader) LoadDerivedChargersFiltered(filter *utils.TPDerivedChargers, save bool) (err error) {
	tpDcses, err := dbr.storDb.GetTpDerivedChargers(filter)
	if err != nil {
		return err
	}
	for _, tpDcs := range tpDcses {
		tag := tpDcs.GetDerivedChargersKey()
		if _, hasIt := dbr.derivedChargers[tag]; !hasIt {
			dbr.derivedChargers[tag] = make(utils.DerivedChargers, 0) // Load object map since we use this method also from LoadDerivedChargers
		}
		for _, tpDc := range tpDcs.DerivedChargers {
			if dc, err := utils.NewDerivedCharger(tpDc.RunId, tpDc.RunFilters, tpDc.ReqTypeField, tpDc.DirectionField, tpDc.TenantField, tpDc.CategoryField,
				tpDc.AccountField, tpDc.SubjectField, tpDc.DestinationField, tpDc.SetupTimeField, tpDc.AnswerTimeField, tpDc.UsageField, tpDc.SupplierField); err != nil {
				return err
			} else {
				dbr.derivedChargers[tag] = append(dbr.derivedChargers[tag], dc)
			}
		}
	}
	if save {
		for dcsKey, dcs := range dbr.derivedChargers {
			if err := dbr.accountDb.SetDerivedChargers(dcsKey, dcs); err != nil {
				return err
			}
		}
	}
	return nil // Placeholder for now
}

func (dbr *DbReader) LoadCdrStatsByTag(tag string, save bool) error {
	storStats, err := dbr.storDb.GetTpCdrStats(dbr.tpid, tag)
	if err != nil {
		return err
	}
	if save && len(dbr.actionsTriggers) == 0 {
		// load action triggers to check existence
		dbr.LoadActionTriggers()
	}
	var loadedTags []string
	for tag, tpStats := range storStats {
		for _, tpStat := range tpStats {
			var cs *CdrStats
			var exists bool
			if cs, exists = dbr.cdrStats[tag]; !exists {
				cs = &CdrStats{Id: tag}
			}
			triggerTag := tpStat.ActionTriggers
			triggers, exists := dbr.actionsTriggers[triggerTag]
			if triggerTag != "" && !exists {
				// only return error if there was something there for the tag
				return fmt.Errorf("Could not get action triggers for cdr stats id %s: %s", cs.Id, triggerTag)
			}
			UpdateCdrStats(cs, triggers, tpStat)
			dbr.cdrStats[tag] = cs
			loadedTags = append(loadedTags, tag)
		}
	}
	if save {
		for _, tag := range loadedTags {
			if err := dbr.dataDb.SetCdrStats(dbr.cdrStats[tag], nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (dbr *DbReader) LoadCdrStats() error {
	return dbr.LoadCdrStatsByTag("", false)
}

// Automated loading
func (dbr *DbReader) LoadAll() error {
	var err error
	if err = dbr.LoadDestinations(); err != nil {
		return err
	}
	if err = dbr.LoadTimings(); err != nil {
		return err
	}
	if err = dbr.LoadRates(); err != nil {
		return err
	}
	if err = dbr.LoadDestinationRates(); err != nil {
		return err
	}
	if err = dbr.LoadRatingPlans(); err != nil {
		return err
	}
	if err = dbr.LoadRatingProfiles(); err != nil {
		return err
	}
	if err = dbr.LoadSharedGroups(); err != nil {
		return err
	}
	if err = dbr.LoadActions(); err != nil {
		return err
	}
	if err = dbr.LoadActionTimings(); err != nil {
		return err
	}
	if err = dbr.LoadActionTriggers(); err != nil {
		return err
	}
	if err = dbr.LoadAccountActions(); err != nil {
		return err
	}
	if err = dbr.LoadDerivedChargers(); err != nil {
		return err
	}
	return nil
}

// Returns the identities loaded for a specific entity category
func (dbr *DbReader) GetLoadedIds(categ string) ([]string, error) {
	switch categ {
	case DESTINATION_PREFIX:
		ids := make([]string, len(dbr.destinations))
		i := 0
		for k := range dbr.destinations {
			ids[i] = k
			i++
		}
		return ids, nil
	case RATING_PLAN_PREFIX:
		keys := make([]string, len(dbr.ratingPlans))
		i := 0
		for k := range dbr.ratingPlans {
			keys[i] = k
			i++
		}
		return keys, nil
	case RATING_PROFILE_PREFIX:
		keys := make([]string, len(dbr.ratingProfiles))
		i := 0
		for k := range dbr.ratingProfiles {
			keys[i] = k
			i++
		}
		return keys, nil
	case ACTION_PREFIX: // actions
		keys := make([]string, len(dbr.actions))
		i := 0
		for k := range dbr.actions {
			keys[i] = k
			i++
		}
		return keys, nil
	case ACTION_TIMING_PREFIX: // actionsTimings
		keys := make([]string, len(dbr.actionsTimings))
		i := 0
		for k := range dbr.actionsTimings {
			keys[i] = k
			i++
		}
		return keys, nil
	case RP_ALIAS_PREFIX: // aliases
		keys := make([]string, len(dbr.rpAliases))
		i := 0
		for k := range dbr.rpAliases {
			keys[i] = k
			i++
		}
		return keys, nil
	case ACC_ALIAS_PREFIX: // aliases
		keys := make([]string, len(dbr.accAliases))
		i := 0
		for k := range dbr.accAliases {
			keys[i] = k
			i++
		}
		return keys, nil
	case DERIVEDCHARGERS_PREFIX: // derived charges
		keys := make([]string, len(dbr.derivedChargers))
		i := 0
		for k := range dbr.derivedChargers {
			keys[i] = k
			i++
		}
		return keys, nil
	case SHARED_GROUP_PREFIX:
		keys := make([]string, len(dbr.sharedGroups))
		i := 0
		for k := range dbr.sharedGroups {
			keys[i] = k
			i++
		}
		return keys, nil
	case CDR_STATS_PREFIX: // cdr stats
		keys := make([]string, len(dbr.cdrStats))
		i := 0
		for k := range dbr.cdrStats {
			keys[i] = k
			i++
		}
		return keys, nil
	}
	return nil, errors.New("Unsupported category")
}
