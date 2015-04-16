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
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cgrates/cgrates/utils"
)

// Define here fields within utils.ACTIONS_CSV file
const (
	ACTSCSVIDX_TAG = iota
	ACTSCSVIDX_ACTION
	ACTSCSVIDX_EXTRA_PARAMS
	ACTSCSVIDX_BALANCE_TAG
	ACTSCSVIDX_BALANCE_TYPE
	ACTSCSVIDX_DIRECTION
	ACTSCSVIDX_CATEGORY
	ACTSCSVIDX_DESTINATION_TAG
	ACTSCSVIDX_RATING_SUBJECT
	ACTSCSVIDX_SHARED_GROUP
	ACTSCSVIDX_EXPIRY_TIME
	ACTSCSVIDX_TIMING_TAGS
	ACTSCSVIDX_UNITS
	ACTSCSVIDX_BALANCE_WEIGHT
	ACTSCSVIDX_WEIGHT
)

// Define here fields within utils.ACTION_TRIGGERS_CSV file
const (
	ATRIGCSVIDX_TAG = iota
	ATRIGCSVIDX_UNIQUE_ID
	ATRIGCSVIDX_THRESHOLD_TYPE
	ATRIGCSVIDX_THRESHOLD_VALUE
	ATRIGCSVIDX_RECURRENT
	ATRIGCSVIDX_MIN_SLEEP
	ATRIGCSVIDX_BAL_TAG
	ATRIGCSVIDX_BAL_TYPE
	ATRIGCSVIDX_BAL_DIRECTION
	ATRIGCSVIDX_BAL_CATEGORY
	ATRIGCSVIDX_BAL_DESTINATION_TAG
	ATRIGCSVIDX_BAL_RATING_SUBJECT
	ATRIGCSVIDX_BAL_SHARED_GROUP
	ATRIGCSVIDX_BAL_EXPIRY_TIME
	ATRIGCSVIDX_BAL_TIMING_TAGS
	ATRIGCSVIDX_BAL_WEIGHT
	ATRIGCSVIDX_STATS_MIN_QUEUED_ITEMS
	ATRIGCSVIDX_ACTIONS_TAG
	ATRIGCSVIDX_WEIGHT
)

type TPLoader interface {
	LoadDestinations() error
	LoadRates() error
	LoadDestinationRates() error
	LoadTimings() error
	LoadRatingPlans() error
	LoadRatingProfiles() error
	LoadSharedGroups() error
	LoadActions() error
	LoadActionTimings() error
	LoadActionTriggers() error
	LoadAccountActions() error
	LoadDerivedChargers() error
	LoadAll() error
	GetLoadedIds(string) ([]string, error)
	ShowStatistics()
	WriteToDatabase(bool, bool) error
	WriteRating(io.Writer, bool, bool) error
	WriteAccounting(io.Writer, bool) error
}

func NewLoadRate(tag, connectFee, price, ratedUnits, rateIncrements, groupInterval string) (r *utils.TPRate, err error) {
	cf, err := strconv.ParseFloat(connectFee, 64)
	if err != nil {
		log.Printf("Error parsing connect fee from: %v", connectFee)
		return
	}
	p, err := strconv.ParseFloat(price, 64)
	if err != nil {
		log.Printf("Error parsing price from: %v", price)
		return
	}

	rs, err := utils.NewRateSlot(cf, p, ratedUnits, rateIncrements, groupInterval)
	if err != nil {
		return nil, err
	}
	r = &utils.TPRate{
		RateId:    tag,
		RateSlots: []*utils.RateSlot{rs},
	}
	return
}

func ValidNextGroup(present, next *utils.RateSlot) error {
	if next.GroupIntervalStartDuration() <= present.GroupIntervalStartDuration() {
		return errors.New(fmt.Sprintf("Next rate group interval start: %#v must be heigher than the previous one: %#v", next, present))
	}
	if math.Mod(next.GroupIntervalStartDuration().Seconds(), present.RateIncrementDuration().Seconds()) != 0 {
		return errors.New(fmt.Sprintf("GroupIntervalStart of %#v must be a multiple of RateIncrement of %#v", next, present))
	}
	return nil
}

func NewTiming(timingInfo ...string) (rt *utils.TPTiming) {
	rt = &utils.TPTiming{}
	rt.Id = timingInfo[0]
	rt.Years.Parse(timingInfo[1], utils.INFIELD_SEP)
	rt.Months.Parse(timingInfo[2], utils.INFIELD_SEP)
	rt.MonthDays.Parse(timingInfo[3], utils.INFIELD_SEP)
	rt.WeekDays.Parse(timingInfo[4], utils.INFIELD_SEP)
	times := strings.Split(timingInfo[5], utils.INFIELD_SEP)
	rt.StartTime = times[0]
	if len(times) > 1 {
		rt.EndTime = times[1]
	}
	return
}

func UpdateCdrStats(cs *CdrStats, triggers ActionTriggerPriotityList, tpCs *utils.TPCdrStat) {
	if tpCs.QueueLength != "" {
		if qi, err := strconv.Atoi(tpCs.QueueLength); err == nil {
			cs.QueueLength = qi
		} else {
			log.Printf("Error parsing QueuedLength %v for cdrs stats %v", tpCs.QueueLength, cs.Id)
		}
	}
	if tpCs.TimeWindow != "" {
		if d, err := time.ParseDuration(tpCs.TimeWindow); err == nil {
			cs.TimeWindow = d
		} else {
			log.Printf("Error parsing TimeWindow %v for cdrs stats %v", tpCs.TimeWindow, cs.Id)
		}
	}
	if tpCs.Metrics != "" {
		cs.Metrics = append(cs.Metrics, tpCs.Metrics)
	}
	if tpCs.SetupInterval != "" {
		times := strings.Split(tpCs.SetupInterval, utils.INFIELD_SEP)
		if len(times) > 0 {
			if sTime, err := utils.ParseTimeDetectLayout(times[0]); err == nil {
				if len(cs.SetupInterval) < 1 {
					cs.SetupInterval = append(cs.SetupInterval, sTime)
				} else {
					cs.SetupInterval[0] = sTime
				}
			} else {
				log.Printf("Error parsing TimeWindow %v for cdrs stats %v", tpCs.SetupInterval, cs.Id)
			}
		}
		if len(times) > 1 {
			if eTime, err := utils.ParseTimeDetectLayout(times[1]); err == nil {
				if len(cs.SetupInterval) < 2 {
					cs.SetupInterval = append(cs.SetupInterval, eTime)
				} else {
					cs.SetupInterval[1] = eTime
				}
			} else {
				log.Printf("Error parsing TimeWindow %v for cdrs stats %v", tpCs.SetupInterval, cs.Id)
			}
		}
	}
	if tpCs.TOR != "" {
		cs.TOR = append(cs.TOR, tpCs.TOR)
	}
	if tpCs.CdrHost != "" {
		cs.CdrHost = append(cs.CdrHost, tpCs.CdrHost)
	}
	if tpCs.CdrSource != "" {
		cs.CdrSource = append(cs.CdrSource, tpCs.CdrSource)
	}
	if tpCs.ReqType != "" {
		cs.ReqType = append(cs.ReqType, tpCs.ReqType)
	}
	if tpCs.Direction != "" {
		cs.Direction = append(cs.Direction, tpCs.Direction)
	}
	if tpCs.Tenant != "" {
		cs.Tenant = append(cs.Tenant, tpCs.Tenant)
	}
	if tpCs.Category != "" {
		cs.Category = append(cs.Category, tpCs.Category)
	}
	if tpCs.Account != "" {
		cs.Account = append(cs.Account, tpCs.Account)
	}
	if tpCs.Subject != "" {
		cs.Subject = append(cs.Subject, tpCs.Subject)
	}
	if tpCs.DestinationPrefix != "" {
		cs.DestinationPrefix = append(cs.DestinationPrefix, tpCs.DestinationPrefix)
	}
	if tpCs.UsageInterval != "" {
		durations := strings.Split(tpCs.UsageInterval, utils.INFIELD_SEP)
		if len(durations) > 0 {
			if sDuration, err := time.ParseDuration(durations[0]); err == nil {
				if len(cs.UsageInterval) < 1 {
					cs.UsageInterval = append(cs.UsageInterval, sDuration)
				} else {
					cs.UsageInterval[0] = sDuration
				}
			} else {
				log.Printf("Error parsing UsageInterval %v for cdrs stats %v", tpCs.UsageInterval, cs.Id)
			}
		}
		if len(durations) > 1 {
			if eDuration, err := time.ParseDuration(durations[1]); err == nil {
				if len(cs.UsageInterval) < 2 {
					cs.UsageInterval = append(cs.UsageInterval, eDuration)
				} else {
					cs.UsageInterval[1] = eDuration
				}
			} else {
				log.Printf("Error parsing UsageInterval %v for cdrs stats %v", tpCs.UsageInterval, cs.Id)
			}
		}
	}
	if tpCs.MediationRunIds != "" {
		cs.MediationRunIds = append(cs.MediationRunIds, tpCs.MediationRunIds)
	}
	if tpCs.RatedAccount != "" {
		cs.RatedAccount = append(cs.RatedAccount, tpCs.RatedAccount)
	}
	if tpCs.RatedSubject != "" {
		cs.RatedSubject = append(cs.RatedSubject, tpCs.RatedSubject)
	}
	if tpCs.CostInterval != "" {
		costs := strings.Split(tpCs.CostInterval, utils.INFIELD_SEP)
		if len(costs) > 0 {
			if sCost, err := strconv.ParseFloat(costs[0], 64); err == nil {
				if len(cs.CostInterval) < 1 {
					cs.CostInterval = append(cs.CostInterval, sCost)
				} else {
					cs.CostInterval[0] = sCost
				}
			} else {
				log.Printf("Error parsing CostInterval %v for cdrs stats %v", tpCs.CostInterval, cs.Id)
			}
		}
		if len(costs) > 1 {
			if eCost, err := strconv.ParseFloat(costs[1], 64); err == nil {
				if len(cs.CostInterval) < 2 {
					cs.CostInterval = append(cs.CostInterval, eCost)
				} else {
					cs.CostInterval[1] = eCost
				}
			} else {
				log.Printf("Error parsing CostInterval %v for cdrs stats %v", tpCs.CostInterval, cs.Id)
			}
		}
	}
	if triggers != nil {
		cs.Triggers = append(cs.Triggers, triggers...)
	}
}

func NewRatingPlan(timing *utils.TPTiming, weight string) (drt *utils.TPRatingPlanBinding) {
	w, err := strconv.ParseFloat(weight, 64)
	if err != nil {
		log.Printf("Error parsing weight unit from: %v", weight)
		return
	}
	drt = &utils.TPRatingPlanBinding{
		Weight: w,
	}
	drt.SetTiming(timing)
	return
}

func GetRateInterval(rpl *utils.TPRatingPlanBinding, dr *utils.DestinationRate) (i *RateInterval) {
	i = &RateInterval{
		Timing: &RITiming{
			Years:     rpl.Timing().Years,
			Months:    rpl.Timing().Months,
			MonthDays: rpl.Timing().MonthDays,
			WeekDays:  rpl.Timing().WeekDays,
			StartTime: rpl.Timing().StartTime,
		},
		Weight: rpl.Weight,
		Rating: &RIRate{
			ConnectFee:       dr.Rate.RateSlots[0].ConnectFee,
			RoundingMethod:   dr.RoundingMethod,
			RoundingDecimals: dr.RoundingDecimals,
			MaxCost:          dr.MaxCost,
			MaxCostStrategy:  dr.MaxCostStrategy,
		},
	}
	for _, rl := range dr.Rate.RateSlots {
		i.Rating.Rates = append(i.Rating.Rates, &Rate{
			GroupIntervalStart: rl.GroupIntervalStartDuration(),
			Value:              rl.Rate,
			RateIncrement:      rl.RateIncrementDuration(),
			RateUnit:           rl.RateUnitDuration(),
		})
	}
	return
}

type AccountAction struct {
	Tenant, Account, Direction, ActionTimingsTag, ActionTriggersTag string
}

func ValidateCSVData(fn string, re *regexp.Regexp) (err error) {
	fin, err := os.Open(fn)
	if err != nil {
		// do not return the error, the file might be not needed
		return nil
	}
	defer fin.Close()
	r := bufio.NewReader(fin)
	line_number := 1
	for {
		line, truncated, err := r.ReadLine()
		if err != nil {
			break
		}
		if truncated {
			return errors.New("line too long")
		}
		// skip the header line
		if line_number > 1 {
			if !re.Match(line) {
				return errors.New(fmt.Sprintf("%s: error on line %d: %s", fn, line_number, line))
			}
		}
		line_number++
	}
	return
}

type FileLineRegexValidator struct {
	FieldsPerRecord int            // Number of fields in one record, useful for crosschecks
	Rule            *regexp.Regexp // Regexp rule
	Message         string         // Pass this message as helper
}

var FileValidators = map[string]*FileLineRegexValidator{
	utils.DESTINATIONS_CSV: &FileLineRegexValidator{utils.DESTINATIONS_NRCOLS,
		regexp.MustCompile(`(?:\w+\s*,\s*){1}(?:\+?\d+.?\d*){1}$`),
		"Tag([0-9A-Za-z_]),Prefix([0-9])"},
	utils.TIMINGS_CSV: &FileLineRegexValidator{utils.TIMINGS_NRCOLS,
		regexp.MustCompile(`(?:\w+\s*,\s*){1}(?:\*any\s*,\s*|(?:\d{1,4};?)+\s*,\s*|\s*,\s*){4}(?:\d{2}:\d{2}:\d{2}|\*asap){1}$`),
		"Tag([0-9A-Za-z_]),Years([0-9;]|*any|<empty>),Months([0-9;]|*any|<empty>),MonthDays([0-9;]|*any|<empty>),WeekDays([0-9;]|*any|<empty>),Time([0-9:]|*asap)"},
	utils.RATES_CSV: &FileLineRegexValidator{utils.RATES_NRCOLS,
		regexp.MustCompile(`(?:\w+\s*),(?:\d+\.*\d*s*),(?:\d+\.*\d*s*),(?:\d+\.*\d*(ns|us|µs|ms|s|m|h)*\s*),(?:\d+\.*\d*(ns|us|µs|ms|s|m|h)*\s*),(?:\d+\.*\d*(ns|us|µs|ms|s|m|h)*\s*)$`),
		"Tag([0-9A-Za-z_]),ConnectFee([0-9.]),Rate([0-9.]),RateUnit([0-9.]ns|us|µs|ms|s|m|h),RateIncrementStart([0-9.]ns|us|µs|ms|s|m|h),GroupIntervalStart([0-9.]ns|us|µs|ms|s|m|h)"},
	utils.DESTINATION_RATES_CSV: &FileLineRegexValidator{utils.DESTINATION_RATES_NRCOLS,
		regexp.MustCompile(`^(?:\w+\s*),(?:\w+\s*|\*any),(?:\w+\s*),(?:\*up|\*down|\*middle),(?:\d+),(?:\d+\.*\d*s*)?,(?:\*free|\*diconnect)?$`),
		"Tag([0-9A-Za-z_]),DestinationsTag([0-9A-Za-z_]|*any),RatesTag([0-9A-Za-z_]),RoundingMethod(*up|*middle|*down),RoundingDecimals([0-9.])"},
	utils.RATING_PLANS_CSV: &FileLineRegexValidator{utils.DESTRATE_TIMINGS_NRCOLS,
		regexp.MustCompile(`(?:\w+\s*,\s*){3}(?:\d+.?\d*){1}$`),
		"Tag([0-9A-Za-z_]),DestinationRatesTag([0-9A-Za-z_]),TimingProfile([0-9A-Za-z_]),Weight([0-9.])"},
	utils.RATING_PROFILES_CSV: &FileLineRegexValidator{utils.RATE_PROFILES_NRCOLS,
		regexp.MustCompile(`^(?:\*out\s*),(?:[0-9A-Za-z_\.]+\s*),(?:\w+\s*),(?:\*any\s*|(\w+;?)+\s*),(?:\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z),(?:\w+\s*),(?:\w+\s*)?,(?:\w+\s*)?$`),
		"Direction(*out),Tenant([0-9A-Za-z_]),Category([0-9A-Za-z_]),Subject([0-9A-Za-z_]|*any),ActivationTime([0-9T:X]),RatingPlanId([0-9A-Za-z_]),RatesFallbackSubject([0-9A-Za-z_]|<empty>),CdrStatQueueIds([0-9A-Za-z_]|<empty>)"},
	utils.SHARED_GROUPS_CSV: &FileLineRegexValidator{utils.SHARED_GROUPS_NRCOLS,
		regexp.MustCompile(`(?:\w+\s*),(?:\*?\w+\s*),(?:\*\w+\s*),(?:\*?\w]+\s*)?`),
		"Id([0-9A-Za-z_]),Account(*?[0-9A-Za-z_]),Strategy(*[0-9A-Za-z_]),RatingSubject(*?[0-9A-Za-z_])"},
	utils.ACTIONS_CSV: &FileLineRegexValidator{utils.ACTIONS_NRCOLS,
		regexp.MustCompile(`^(?:\w+\s*),(?:\*\w+\s*),(?:\S+\s*)?,(?:\w+\s*)?,(?:\*\w+\s*)?,(?:\*out\s*)?,(?:\*?\w+\s*)?,(?:\*any|\w+\s*)?,(?:\w+\s*)?,(?:\w+\s*)?,(?:\*\w+\s*|\+\d+[smh]\s*|\d+\s*)?,(?:[0-9A-Za-z_;]*)?,(?:\d+\s*)?,(?:\d+\.?\d*\s*)?,(?:\d+\.?\d*\s*)$`),
		"Tag([0-9A-Za-z_]),Action([0-9A-Za-z_]),ExtraParameters([0-9A-Za-z_:;]),BalanceTag([0-9A-Za-z_]),BalanceType([*a-z_]),Direction(*out),Category([0-9A-Za-z_]),DestinationTag([0-9A-Za-z_]|*any),RatingSubject([0-9A-Za-z_]),SharedGroup([0-9A-Za-z_]),ExpiryTime(*[a-z_]|+[0-9][smh]|[0-9]),TimingTags(([0-9A-Za-z_];?)*),Units([0-9]),BalanceWeight([0-9.]),Weight([0-9.])"},
	utils.ACTION_PLANS_CSV: &FileLineRegexValidator{utils.ACTION_PLANS_NRCOLS,
		regexp.MustCompile(`(?:\w+\s*,\s*){3}(?:\d+\.?\d*){1}`),
		"Tag([0-9A-Za-z_]),ActionsTag([0-9A-Za-z_]),TimingTag([0-9A-Za-z_]),Weight([0-9.])"},
	utils.ACTION_TRIGGERS_CSV: &FileLineRegexValidator{utils.ACTION_TRIGGERS_NRCOLS, regexp.MustCompile(`(?:\w+),(?:\w+)?,(?:\*\w+),(?:\d+\.?\d*),(?:true|false)?,(?:\d+[smh]?),(?:\w+\s*)?,(?:\*\w+)?,(?:\*out)?,(?:\w+|\*any)?,(?:\w+|\*any)?,(?:\w+|\*any)?,(?:\w+|\*any)?,(?:\*\w+\s*|\+\d+[smh]\s*|\d+\s*)?,(?:[0-9A-Za-z_;]*)?,(?:\d+\.?\d*)?,(?:\d+)?,(?:\w+),(?:\d+\.?\d*)$`),
		"Tag([0-9A-Za-z_]),UniqueId([0-9A-Za-z_]),ThresholdType(*[a-z_]),ThresholdValue([0-9]+),Recurrent(true|false),MinSleep([0-9]+)?,BalanceTag([0-9A-Za-z_]),BalanceType(*[a-z_]),BalanceDirection(*out),BalanceCategory([a-z_]),BalanceDestinationTag([0-9A-Za-z_]|*all),BalanceRatingSubject(*[a-z_]),BalanceSharedGroup(*[a-z_]),BalanceExpiryTime(*[a-z_]|+[0-9][smh]|[0-9]),BalanceTimingTags(([0-9A-Za-z_];?)*)BalanceWeight(*[a-z_]),StatsMinQueuedItems([0-9]+),ActionsTag([0-9A-Za-z_]),Weight([0-9]+)"},
	utils.ACCOUNT_ACTIONS_CSV: &FileLineRegexValidator{utils.ACCOUNT_ACTIONS_NRCOLS,
		regexp.MustCompile(`(?:\w+\s*),(?:(\w+;?)+\s*),(?:\*out\s*),(?:\w+\s*),(?:\w+\s*)$`),
		"Tenant([0-9A-Za-z_]),Account([0-9A-Za-z_.]),Direction(*out),ActionTimingsTag([0-9A-Za-z_]),ActionTriggersTag([0-9A-Za-z_])"},
	utils.DERIVED_CHARGERS_CSV: &FileLineRegexValidator{utils.DERIVED_CHARGERS_NRCOLS,
		regexp.MustCompile(`^(?:\*out),(?:[0-9A-Za-z_\.]+\s*),(?:\w+\s*),(?:\w+\s*),(?:\*any\s*|\w+\s*),(?:\w+\s*),(?:[~^]*[0-9A-Za-z_/:().+]+\s*)?,(?:\*default\s*|[~^*]*[0-9A-Za-z_/:().+]+\s*)?,(?:\*default\s*|[~^]*[0-9A-Za-z_/:().+]+\s*)?,(?:\*default\s*|[~^]*[0-9A-Za-z_/:().+]+\s*)?,(?:\*default\s*|[~^]*[0-9A-Za-z_/:().+]+\s*)?,(?:\*default\s*|[~^]*[0-9A-Za-z_/:().+]+\s*)?,(?:\*default\s*|[~^]*[0-9A-Za-z_/:().+]+\s*)?,(?:\*default\s*|[~^]*[0-9A-Za-z_/:().+]+\s*)?,(?:\*default\s*|[~^]*[0-9A-Za-z_/:().+]+\s*)?,(?:\*default\s*|[~^]*[0-9A-Za-z_/:().+]+\s*)?,(?:\*default\s*|[~^]*[0-9A-Za-z_/:().+]+\s*)?,(?:\*default\s*|[~^]*[0-9A-Za-z_/:().+]+\s*)?$`),
		"Direction(*out),Tenant[0-9A-Za-z_],Category([0-9A-Za-z_]),Account[0-9A-Za-z_],Subject([0-9A-Za-z_]|*any),RunId([0-9A-Za-z_]),RunFilter([^~]*[0-9A-Za-z_/]),ReqTypeField([^~]*[0-9A-Za-z_/]|*default),DirectionField([^~]*[0-9A-Za-z_/]|*default),TenantField([^~]*[0-9A-Za-z_/]|*default),TorField([^~]*[0-9A-Za-z_/]|*default),AccountField([^~]*[0-9A-Za-z_/]|*default),SubjectField([^~]*[0-9A-Za-z_/]|*default),DestinationField([^~]*[0-9A-Za-z_/]|*default),SetupTimeField([^~]*[0-9A-Za-z_/]|*default),AnswerTimeField([^~]*[0-9A-Za-z_/]|*default),UsageField([^~]*[0-9A-Za-z_/]|*default),SupplierField([^~]*[0-9A-Za-z_/]|*default)"},
	utils.CDR_STATS_CSV: &FileLineRegexValidator{utils.CDR_STATS_NRCOLS,
		regexp.MustCompile(`.+`), //ToDo: Fix me with proper rules
		"Id,QueueLength,TimeWindow,Metric,SetupInterval,TOR,CdrHost,CdrSource,ReqType,Direction,Tenant,Category,Account,Subject,DestinationPrefix,UsageInterval,MediationRunIds,RatedAccount,RatedSubject,CostInterval,Triggers(*?[0-9A-Za-z_]),Strategy(*[0-9A-Za-z_]),RatingSubject(*?[0-9A-Za-z_])"},
}

func NewTPCSVFileParser(dirPath, fileName string) (*TPCSVFileParser, error) {
	validator, hasValidator := FileValidators[fileName]
	if !hasValidator {
		return nil, fmt.Errorf("No validator found for file <%s>", fileName)
	}
	// Open the file here
	fin, err := os.Open(path.Join(dirPath, fileName))
	if err != nil {
		return nil, err
	}
	//defer fin.Close()
	reader := bufio.NewReader(fin)
	return &TPCSVFileParser{validator, reader}, nil
}

// Opens the connection to a file and returns the parsed lines one by one when ParseNextLine() is called
type TPCSVFileParser struct {
	validator *FileLineRegexValidator // Row validator
	reader    *bufio.Reader           // Reader to the file we are interested in
}

func (self *TPCSVFileParser) ParseNextLine() ([]string, error) {
	line, truncated, err := self.reader.ReadLine()
	if err != nil {
		return nil, err
	} else if truncated {
		return nil, errors.New("Line too long.")
	}
	// skip commented lines
	if strings.HasPrefix(string(line), string(utils.COMMENT_CHAR)) {
		return nil, errors.New("Line starts with comment character.")
	}
	// Validate here string line
	if !self.validator.Rule.Match(line) {
		return nil, fmt.Errorf("Invalid line, <%s>", self.validator.Message)
	}
	// Open csv reader directly on string line
	csvReader, _, err := openStringCSVReader(string(line), ',', self.validator.FieldsPerRecord)
	if err != nil {
		return nil, err
	}
	record, err := csvReader.Read() // if no errors, record should be good to go having right format and length
	if err != nil {
		return nil, err
	}
	return record, nil
}

// Used to populate empty values with *any or *default if value missing
func ValueOrDefault(val string, deflt string) string {
	if len(val) == 0 {
		val = deflt
	}
	return val
}
