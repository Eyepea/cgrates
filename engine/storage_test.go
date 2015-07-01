/*
Rating system designed to be used in VoIP Carriers World
Copyright (C) 2012-2015 ITsysCOM

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITH*out ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package engine

import (
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/cgrates/cgrates/cache2go"
	"github.com/cgrates/cgrates/utils"
)

func TestMsgpackStructsAdded(t *testing.T) {
	var a = struct{ First string }{"test"}
	var b = struct {
		First  string
		Second string
	}{}
	m := new(MsgpackMarshaler)
	buf, err := m.Marshal(&a)
	if err != nil {
		t.Error("error marshaling structure: ", err)
	}
	err = m.Unmarshal(buf, &b)
	if err != nil || b.First != "test" || b.Second != "" {
		t.Error("error unmarshalling structure: ", b, err)
	}
}

func TestMsgpackStructsMissing(t *testing.T) {
	var a = struct {
		First  string
		Second string
	}{"test1", "test2"}
	var b = struct{ First string }{}
	m := new(MsgpackMarshaler)
	buf, err := m.Marshal(&a)
	if err != nil {
		t.Error("error marshaling structure: ", err)
	}
	err = m.Unmarshal(buf, &b)
	if err != nil || b.First != "test1" {
		t.Error("error unmarshalling structure: ", b, err)
	}
}

func TestMsgpackTime(t *testing.T) {
	t1 := time.Date(2013, 8, 28, 22, 27, 0, 0, time.UTC)
	m := new(MsgpackMarshaler)
	buf, err := m.Marshal(&t1)
	if err != nil {
		t.Error("error marshaling structure: ", err)
	}
	var t2 time.Time
	err = m.Unmarshal(buf, &t2)
	if err != nil || !t1.Equal(t2) {
		t.Errorf("error unmarshalling structure: %#v %#v %v", t1, t2, err)
	}
}

func TestStorageDestinationContainsPrefixShort(t *testing.T) {
	dest, err := ratingStorage.GetDestination("NAT")
	precision := dest.containsPrefix("0723")
	if err != nil || precision != 4 {
		t.Error("Error finding prefix: ", err, precision)
	}
}

func TestStorageDestinationContainsPrefixLong(t *testing.T) {
	dest, err := ratingStorage.GetDestination("NAT")
	precision := dest.containsPrefix("0723045326")
	if err != nil || precision != 4 {
		t.Error("Error finding prefix: ", err, precision)
	}
}

func TestStorageDestinationContainsPrefixNotExisting(t *testing.T) {
	dest, err := ratingStorage.GetDestination("NAT")
	precision := dest.containsPrefix("072")
	if err != nil || precision != 0 {
		t.Error("Error finding prefix: ", err, precision)
	}
}

func TestCacheRefresh(t *testing.T) {
	ratingStorage.SetDestination(&Destination{"T11", []string{"0"}})
	ratingStorage.GetDestination("T11")
	ratingStorage.SetDestination(&Destination{"T11", []string{"1"}})
	t.Log("Test cache refresh")
	ratingStorage.CacheAll()
	d, err := ratingStorage.GetDestination("T11")
	p := d.containsPrefix("1")
	if err != nil || p == 0 {
		t.Error("Error refreshing cache:", d)
	}
}

func TestCacheAliases(t *testing.T) {
	if subj, err := cache2go.GetCached(utils.RP_ALIAS_PREFIX + utils.RatingSubjectAliasKey("vdf", "a3")); err == nil && subj != "minu" {
		t.Error("Error caching alias: ", subj, err)
	}
}

// Install fails to detect them and starting server will panic, these tests will fix this
func TestStoreInterfaces(t *testing.T) {
	rds := new(RedisStorage)
	var _ RatingStorage = rds
	var _ AccountingStorage = rds
	sql := new(SQLStorage)
	var _ CdrStorage = sql
	var _ LogStorage = sql
}

func TestGetRPAliases(t *testing.T) {
	if err := ratingStorage.SetRpAlias(utils.RatingSubjectAliasKey("cgrates.org", "2001"), "1001"); err != nil {
		t.Error(err)
	}
	if err := ratingStorage.SetRpAlias(utils.RatingSubjectAliasKey("cgrates.org", "2002"), "1001"); err != nil {
		t.Error(err)
	}
	if err := ratingStorage.SetRpAlias(utils.RatingSubjectAliasKey("itsyscom.com", "2003"), "1001"); err != nil {
		t.Error(err)
	}
	expectAliases := sort.StringSlice([]string{"2001", "2002"})
	expectAliases.Sort()
	if aliases, err := ratingStorage.GetRPAliases("cgrates.org", "1001", true); err != nil {
		t.Error(err)
	} else {
		aliases := sort.StringSlice(aliases)
		aliases.Sort()
		if !reflect.DeepEqual(aliases, expectAliases) {
			t.Errorf("Expecting: %v, received: %v", expectAliases, aliases)
		}
	}
}

func TestRemRSubjAliases(t *testing.T) {
	if err := ratingStorage.SetRpAlias(utils.RatingSubjectAliasKey("cgrates.org", "2001"), "1001"); err != nil {
		t.Error(err)
	}
	if err := ratingStorage.SetRpAlias(utils.RatingSubjectAliasKey("cgrates.org", "2002"), "1001"); err != nil {
		t.Error(err)
	}
	if err := ratingStorage.SetRpAlias(utils.RatingSubjectAliasKey("itsyscom.com", "2003"), "1001"); err != nil {
		t.Error(err)
	}
	if err := ratingStorage.RemoveRpAliases([]*TenantRatingSubject{&TenantRatingSubject{Tenant: "cgrates.org", Subject: "1001"}}); err != nil {
		t.Error(err)
	}
	if cgrAliases, err := ratingStorage.GetRPAliases("cgrates.org", "1001", true); err != nil {
		t.Error(err)
	} else if len(cgrAliases) != 0 {
		t.Error("Subject aliases not removed: ", cgrAliases)
	}
	if iscAliases, err := ratingStorage.GetRPAliases("itsyscom.com", "1001", true); err != nil { // Make sure the aliases were removed at tenant level
		t.Error(err)
	} else if !reflect.DeepEqual(iscAliases, []string{"2003"}) {
		t.Errorf("Unexpected aliases: %v", iscAliases)
	}
}

func TestGetAccountAliases(t *testing.T) {
	if err := ratingStorage.SetAccAlias(utils.AccountAliasKey("cgrates.org", "2001"), "1001"); err != nil {
		t.Error(err)
	}
	if err := ratingStorage.SetAccAlias(utils.AccountAliasKey("cgrates.org", "2002"), "1001"); err != nil {
		t.Error(err)
	}
	if err := ratingStorage.SetAccAlias(utils.AccountAliasKey("itsyscom.com", "2003"), "1001"); err != nil {
		t.Error(err)
	}
	expectAliases := sort.StringSlice([]string{"2001", "2002"})
	expectAliases.Sort()
	if aliases, err := ratingStorage.GetAccountAliases("cgrates.org", "1001", true); err != nil {
		t.Error(err)
	} else {
		aliases := sort.StringSlice(aliases)
		aliases.Sort()
		if !reflect.DeepEqual(aliases, expectAliases) {
			t.Errorf("Expecting: %v, received: %v", expectAliases, aliases)
		}
	}
}

/************************** Benchmarks *****************************/

func GetUB() *Account {
	uc := &UnitsCounter{
		Direction:   OUTBOUND,
		BalanceType: utils.SMS,
		Balances:    BalanceChain{&Balance{Value: 1}, &Balance{Weight: 20, DestinationIds: "NAT"}, &Balance{Weight: 10, DestinationIds: "RET"}},
	}
	at := &ActionTrigger{
		Id:                    "some_uuid",
		BalanceType:           utils.MONETARY,
		BalanceDirection:      OUTBOUND,
		ThresholdValue:        100.0,
		BalanceDestinationIds: "NAT",
		Weight:                10.0,
		ActionsId:             "Commando",
	}
	var zeroTime time.Time
	zeroTime = zeroTime.UTC() // for deep equal to find location
	ub := &Account{
		Id:             "rif",
		AllowNegative:  true,
		BalanceMap:     map[string]BalanceChain{utils.SMS + OUTBOUND: BalanceChain{&Balance{Value: 14, ExpirationDate: zeroTime}}, utils.DATA + OUTBOUND: BalanceChain{&Balance{Value: 1024, ExpirationDate: zeroTime}}, utils.VOICE: BalanceChain{&Balance{Weight: 20, DestinationIds: "NAT"}, &Balance{Weight: 10, DestinationIds: "RET"}}},
		UnitCounters:   []*UnitsCounter{uc, uc},
		ActionTriggers: ActionTriggerPriotityList{at, at, at},
	}
	return ub
}

func BenchmarkMarshallerJSONStoreRestore(b *testing.B) {
	b.StopTimer()
	i := &RateInterval{
		Timing: &RITiming{
			Months:    []time.Month{time.February},
			MonthDays: []int{1},
			WeekDays:  []time.Weekday{time.Wednesday, time.Thursday},
			StartTime: "14:30:00",
			EndTime:   "15:00:00"}}
	ap := &RatingPlan{Id: "test"}
	ap.AddRateInterval("NAT", i)
	ub := GetUB()

	ap1 := RatingPlan{}
	ub1 := &Account{}
	b.StartTimer()
	ms := new(JSONMarshaler)
	for i := 0; i < b.N; i++ {
		result, _ := ms.Marshal(ap)
		ms.Unmarshal(result, ap1)
		result, _ = ms.Marshal(ub)
		ms.Unmarshal(result, ub1)
	}
}

func BenchmarkMarshallerBSONStoreRestore(b *testing.B) {
	b.StopTimer()
	i := &RateInterval{
		Timing: &RITiming{
			Months:    []time.Month{time.February},
			MonthDays: []int{1},
			WeekDays:  []time.Weekday{time.Wednesday, time.Thursday},
			StartTime: "14:30:00",
			EndTime:   "15:00:00"}}
	ap := &RatingPlan{Id: "test"}
	ap.AddRateInterval("NAT", i)
	ub := GetUB()

	ap1 := RatingPlan{}
	ub1 := &Account{}
	b.StartTimer()
	ms := new(BSONMarshaler)
	for i := 0; i < b.N; i++ {
		result, _ := ms.Marshal(ap)
		ms.Unmarshal(result, ap1)
		result, _ = ms.Marshal(ub)
		ms.Unmarshal(result, ub1)
	}
}

func BenchmarkMarshallerMsgpackStoreRestore(b *testing.B) {
	b.StopTimer()
	i := &RateInterval{
		Timing: &RITiming{
			Months:    []time.Month{time.February},
			MonthDays: []int{1},
			WeekDays:  []time.Weekday{time.Wednesday, time.Thursday},
			StartTime: "14:30:00",
			EndTime:   "15:00:00"}}
	ap := &RatingPlan{Id: "test"}
	ap.AddRateInterval("NAT", i)
	ub := GetUB()

	ap1 := RatingPlan{}
	ub1 := &Account{}
	b.StartTimer()
	ms := new(MsgpackMarshaler)
	for i := 0; i < b.N; i++ {
		result, _ := ms.Marshal(ap)
		ms.Unmarshal(result, ap1)
		result, _ = ms.Marshal(ub)
		ms.Unmarshal(result, ub1)
	}
}

func BenchmarkMarshallerJSONBufStoreRestore(b *testing.B) {
	b.StopTimer()
	i := &RateInterval{
		Timing: &RITiming{Months: []time.Month{time.February},
			MonthDays: []int{1},
			WeekDays:  []time.Weekday{time.Wednesday, time.Thursday},
			StartTime: "14:30:00",
			EndTime:   "15:00:00"}}
	ap := &RatingPlan{Id: "test"}
	ap.AddRateInterval("NAT", i)
	ub := GetUB()

	ap1 := RatingPlan{}
	ub1 := &Account{}
	b.StartTimer()
	ms := new(JSONBufMarshaler)
	for i := 0; i < b.N; i++ {
		result, _ := ms.Marshal(ap)
		ms.Unmarshal(result, ap1)
		result, _ = ms.Marshal(ub)
		ms.Unmarshal(result, ub1)
	}
}

func BenchmarkMarshallerGOBStoreRestore(b *testing.B) {
	b.StopTimer()
	i := &RateInterval{
		Timing: &RITiming{Months: []time.Month{time.February},
			MonthDays: []int{1},
			WeekDays:  []time.Weekday{time.Wednesday, time.Thursday},
			StartTime: "14:30:00",
			EndTime:   "15:00:00"}}
	ap := &RatingPlan{Id: "test"}
	ap.AddRateInterval("NAT", i)
	ub := GetUB()

	ap1 := RatingPlan{}
	ub1 := &Account{}
	b.StartTimer()
	ms := new(GOBMarshaler)
	for i := 0; i < b.N; i++ {
		result, _ := ms.Marshal(ap)
		ms.Unmarshal(result, ap1)
		result, _ = ms.Marshal(ub)
		ms.Unmarshal(result, ub1)
	}
}

/*
func BenchmarkMarshallerCodecMsgpackStoreRestore(b *testing.B) {
	b.StopTimer()
	i := &RateInterval{
		Timing: &RITiming{
			Months:    []time.Month{time.February},
			MonthDays: []int{1},
			WeekDays:  []time.Weekday{time.Wednesday, time.Thursday},
			StartTime: "14:30:00",
			EndTime:   "15:00:00"}}
	ap := &RatingPlan{Id: "test"}
	ap.AddRateInterval("NAT", i)
	ub := GetUB()

	ap1 := RatingPlan{}
	ub1 := &Account{}
	b.StartTimer()
	ms := NewCodecMsgpackMarshaler()
	for i := 0; i < b.N; i++ {
		result, _ := ms.Marshal(ap)
		ms.Unmarshal(result, ap1)
		result, _ = ms.Marshal(ub)
		ms.Unmarshal(result, ub1)
	}
}

func BenchmarkMarshallerBincStoreRestore(b *testing.B) {
	b.StopTimer()
	i := &RateInterval{
		Timing: &RITiming{
			Months:    []time.Month{time.February},
			MonthDays: []int{1},
			WeekDays:  []time.Weekday{time.Wednesday, time.Thursday},
			StartTime: "14:30:00",
			EndTime:   "15:00:00"}}
	ap := &RatingPlan{Id: "test"}
	ap.AddRateInterval("NAT", i)
	ub := GetUB()

	ap1 := RatingPlan{}
	ub1 := &Account{}
	b.StartTimer()
	ms := NewBincMarshaler()
	for i := 0; i < b.N; i++ {
		result, _ := ms.Marshal(ap)
		ms.Unmarshal(result, ap1)
		result, _ = ms.Marshal(ub)
		ms.Unmarshal(result, ub1)
	}
}
*/
