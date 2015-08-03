/*
Real-time Charging System for Telecom & ISP environments
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
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/cgrates/cgrates/utils"
)

var (
	//referenceDate = time.Date(2013, 7, 10, 10, 30, 0, 0, time.Local)
	//referenceDate = time.Date(2013, 12, 31, 23, 59, 59, 0, time.Local)
	//referenceDate = time.Date(2011, 1, 1, 0, 0, 0, 1, time.Local)
	referenceDate = time.Now()
	now           = referenceDate
)

func TestActionTimingAlways(t *testing.T) {
	at := &ActionPlan{Timing: &RateInterval{Timing: &RITiming{StartTime: "00:00:00"}}}
	st := at.GetNextStartTime(referenceDate)
	y, m, d := referenceDate.Date()
	expected := time.Date(y, m, d, 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
	if !st.Equal(expected) {
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanNothing(t *testing.T) {
	at := &ActionPlan{}
	st := at.GetNextStartTime(referenceDate)
	expected := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	if !st.Equal(expected) {
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanOnlyHour(t *testing.T) {
	at := &ActionPlan{Timing: &RateInterval{Timing: &RITiming{StartTime: "10:01:00"}}}
	st := at.GetNextStartTime(referenceDate)

	y, m, d := now.Date()
	expected := time.Date(y, m, d, 10, 1, 0, 0, time.Local)
	if referenceDate.After(expected) {
		expected = expected.AddDate(0, 0, 1)
	}
	if !st.Equal(expected) {
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanHourYear(t *testing.T) {
	at := &ActionPlan{Timing: &RateInterval{Timing: &RITiming{Years: utils.Years{2022}, StartTime: "10:01:00"}}}
	st := at.GetNextStartTime(referenceDate)
	expected := time.Date(2022, 1, 1, 10, 1, 0, 0, time.Local)
	if !st.Equal(expected) {
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanOnlyWeekdays(t *testing.T) {
	at := &ActionPlan{Timing: &RateInterval{Timing: &RITiming{WeekDays: []time.Weekday{time.Monday}}}}
	st := at.GetNextStartTime(referenceDate)

	y, m, d := now.Date()
	h, min, s := now.Clock()
	e := time.Date(y, m, d, h, min, s, 0, time.Local)
	day := e.Day()
	e = time.Date(e.Year(), e.Month(), day, 0, 0, 0, 0, e.Location())
	for i := 0; i < 8; i++ {
		n := e.AddDate(0, 0, i)
		if n.Weekday() == time.Monday && (n.Equal(now) || n.After(now)) {
			e = n
			break
		}
	}
	if !st.Equal(e) {
		t.Errorf("Expected %v was %v", e, st)
	}
}

func TestActionPlanHourWeekdays(t *testing.T) {
	at := &ActionPlan{Timing: &RateInterval{Timing: &RITiming{WeekDays: []time.Weekday{time.Monday}, StartTime: "10:01:00"}}}
	st := at.GetNextStartTime(referenceDate)

	y, m, d := now.Date()
	e := time.Date(y, m, d, 10, 1, 0, 0, time.Local)
	day := e.Day()
	for i := 0; i < 8; i++ {
		e = time.Date(e.Year(), e.Month(), day, e.Hour(), e.Minute(), e.Second(), e.Nanosecond(), e.Location())
		n := e.AddDate(0, 0, i)
		if n.Weekday() == time.Monday && (n.Equal(now) || n.After(now)) {
			e = n
			break
		}
	}
	if !st.Equal(e) {
		t.Errorf("Expected %v was %v", e, st)
	}
}

func TestActionPlanOnlyMonthdays(t *testing.T) {

	y, m, d := now.Date()
	tomorrow := time.Date(y, m, d, 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
	at := &ActionPlan{Timing: &RateInterval{Timing: &RITiming{MonthDays: utils.MonthDays{1, 25, 2, tomorrow.Day()}}}}
	st := at.GetNextStartTime(referenceDate)
	expected := tomorrow
	if !st.Equal(expected) {
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanHourMonthdays(t *testing.T) {

	y, m, d := now.Date()
	testTime := time.Date(y, m, d, 10, 1, 0, 0, time.Local)
	tomorrow := time.Date(y, m, d, 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
	if now.After(testTime) {
		y, m, d = tomorrow.Date()
	}
	at := &ActionPlan{Timing: &RateInterval{Timing: &RITiming{MonthDays: utils.MonthDays{now.Day(), tomorrow.Day()}, StartTime: "10:01:00"}}}
	st := at.GetNextStartTime(referenceDate)
	expected := time.Date(y, m, d, 10, 1, 0, 0, time.Local)
	if !st.Equal(expected) {
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanOnlyMonths(t *testing.T) {

	y, m, _ := now.Date()
	nextMonth := time.Date(y, m, 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, 0)
	at := &ActionPlan{Timing: &RateInterval{Timing: &RITiming{Months: utils.Months{time.February, time.May, nextMonth.Month()}}}}
	st := at.GetNextStartTime(referenceDate)
	expected := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, time.Local)
	if !st.Equal(expected) {
		t.Log("NextMonth: ", nextMonth)
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanHourMonths(t *testing.T) {

	y, m, d := now.Date()
	testTime := time.Date(y, m, d, 10, 1, 0, 0, time.Local)
	nextMonth := time.Date(y, m, 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, 0)
	if now.After(testTime) {
		testTime = testTime.AddDate(0, 0, 1)
		y, m, d = testTime.Date()
	}
	if now.After(testTime) {
		m = nextMonth.Month()
		y = nextMonth.Year()

	}
	at := &ActionPlan{Timing: &RateInterval{Timing: &RITiming{
		Months:    utils.Months{now.Month(), nextMonth.Month()},
		StartTime: "10:01:00"}}}
	st := at.GetNextStartTime(referenceDate)
	expected := time.Date(y, m, 1, 10, 1, 0, 0, time.Local)
	if referenceDate.After(expected) {
		expected = expected.AddDate(0, 1, 0)
	}
	if !st.Equal(expected) {
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanHourMonthdaysMonths(t *testing.T) {

	y, m, d := now.Date()
	testTime := time.Date(y, m, d, 10, 1, 0, 0, time.Local)
	nextMonth := time.Date(y, m, 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, 0)
	tomorrow := time.Date(y, m, d, 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)

	if now.After(testTime) {
		y, m, d = tomorrow.Date()
	}
	nextDay := time.Date(y, m, d, 10, 1, 0, 0, time.Local)
	month := nextDay.Month()
	if nextDay.Before(now) {
		if now.After(testTime) {
			month = nextMonth.Month()
		}
	}
	at := &ActionPlan{Timing: &RateInterval{
		Timing: &RITiming{
			Months:    utils.Months{now.Month(), nextMonth.Month()},
			MonthDays: utils.MonthDays{now.Day(), tomorrow.Day()},
			StartTime: "10:01:00",
		},
	}}
	st := at.GetNextStartTime(referenceDate)
	expected := time.Date(y, month, d, 10, 1, 0, 0, time.Local)
	if !st.Equal(expected) {
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanFirstOfTheMonth(t *testing.T) {

	y, m, _ := now.Date()
	nextMonth := time.Date(y, m, 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, 0)
	at := &ActionPlan{Timing: &RateInterval{
		Timing: &RITiming{
			MonthDays: utils.MonthDays{1},
		},
	}}
	st := at.GetNextStartTime(referenceDate)
	expected := nextMonth
	if !st.Equal(expected) {
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanOnlyYears(t *testing.T) {
	y, _, _ := referenceDate.Date()
	nextYear := time.Date(y, 1, 1, 0, 0, 0, 0, time.Local).AddDate(1, 0, 0)
	at := &ActionPlan{Timing: &RateInterval{Timing: &RITiming{Years: utils.Years{now.Year(), nextYear.Year()}}}}
	st := at.GetNextStartTime(referenceDate)
	expected := nextYear
	if !st.Equal(expected) {
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanPast(t *testing.T) {
	at := &ActionPlan{Timing: &RateInterval{Timing: &RITiming{Years: utils.Years{2023}}}}
	st := at.GetNextStartTime(referenceDate)
	expected := time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local)
	if !st.Equal(expected) {
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanHourYears(t *testing.T) {
	at := &ActionPlan{Timing: &RateInterval{Timing: &RITiming{Years: utils.Years{referenceDate.Year(), referenceDate.Year() + 1}, StartTime: "10:01:00"}}}
	st := at.GetNextStartTime(referenceDate)
	expected := time.Date(referenceDate.Year(), 1, 1, 10, 1, 0, 0, time.Local)
	if referenceDate.After(expected) {
		expected = expected.AddDate(1, 0, 0)
	}
	if !st.Equal(expected) {
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanHourMonthdaysYear(t *testing.T) {

	y, m, d := now.Date()
	testTime := time.Date(y, m, d, 10, 1, 0, 0, time.Local)
	tomorrow := time.Date(y, m, d, 10, 1, 0, 0, time.Local).AddDate(0, 0, 1)
	nextYear := time.Date(y, 1, d, 10, 1, 0, 0, time.Local).AddDate(1, 0, 0)
	expected := testTime
	if referenceDate.After(testTime) {
		if referenceDate.After(tomorrow) {
			expected = nextYear
		} else {
			expected = tomorrow
		}
	}
	at := &ActionPlan{Timing: &RateInterval{
		Timing: &RITiming{
			Years:     utils.Years{now.Year(), nextYear.Year()},
			MonthDays: utils.MonthDays{now.Day(), tomorrow.Day()},
			StartTime: "10:01:00",
		},
	}}
	t.Log(at.Timing.Timing.CronString())
	t.Log(time.Now(), referenceDate, referenceDate.After(testTime), referenceDate.After(testTime))
	st := at.GetNextStartTime(referenceDate)
	if !st.Equal(expected) {
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanHourMonthdaysMonthYear(t *testing.T) {

	y, m, d := now.Date()
	testTime := time.Date(y, m, d, 10, 1, 0, 0, time.Local)
	nextYear := time.Date(y, m, 1, 0, 0, 0, 0, time.Local).AddDate(1, 0, 0)
	nextMonth := time.Date(y, m, 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, 0)
	tomorrow := time.Date(y, m, d, 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
	day := now.Day()
	if now.After(testTime) {
		day = tomorrow.Day()
	}
	nextDay := time.Date(y, m, day, 10, 1, 0, 0, time.Local)
	month := now.Month()
	if nextDay.Before(now) {
		if now.After(testTime) {
			month = nextMonth.Month()
		}
	}
	nextDay = time.Date(y, month, day, 10, 1, 0, 0, time.Local)
	year := now.Year()
	if nextDay.Before(now) {
		if now.After(testTime) {
			year = nextYear.Year()
		}
	}
	at := &ActionPlan{Timing: &RateInterval{
		Timing: &RITiming{
			Years:     utils.Years{now.Year(), nextYear.Year()},
			Months:    utils.Months{now.Month(), nextMonth.Month()},
			MonthDays: utils.MonthDays{now.Day(), tomorrow.Day()},
			StartTime: "10:01:00",
		},
	}}
	st := at.GetNextStartTime(referenceDate)
	expected := time.Date(year, month, day, 10, 1, 0, 0, time.Local)
	if !st.Equal(expected) {
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanFirstOfTheYear(t *testing.T) {
	y, _, _ := now.Date()
	nextYear := time.Date(y, 1, 1, 0, 0, 0, 0, time.Local).AddDate(1, 0, 0)
	at := &ActionPlan{Timing: &RateInterval{
		Timing: &RITiming{
			Years:     utils.Years{nextYear.Year()},
			Months:    utils.Months{time.January},
			MonthDays: utils.MonthDays{1},
			StartTime: "00:00:00",
		},
	}}
	st := at.GetNextStartTime(referenceDate)
	expected := nextYear
	if !st.Equal(expected) {
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanFirstMonthOfTheYear(t *testing.T) {
	y, _, _ := now.Date()
	expected := time.Date(y, 1, 1, 0, 0, 0, 0, time.Local)
	if referenceDate.After(expected) {
		expected = expected.AddDate(1, 0, 0)
	}
	at := &ActionPlan{Timing: &RateInterval{
		Timing: &RITiming{
			Months: utils.Months{time.January},
		},
	}}
	st := at.GetNextStartTime(referenceDate)
	if !st.Equal(expected) {
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanFirstMonthOfTheYearSecondDay(t *testing.T) {
	y, _, _ := now.Date()
	expected := time.Date(y, 1, 2, 0, 0, 0, 0, time.Local)
	if referenceDate.After(expected) {
		expected = expected.AddDate(1, 0, 0)
	}
	at := &ActionPlan{Timing: &RateInterval{
		Timing: &RITiming{
			Months:    utils.Months{time.January},
			MonthDays: utils.MonthDays{2},
		},
	}}
	st := at.GetNextStartTime(referenceDate)
	if !st.Equal(expected) {
		t.Errorf("Expected %v was %v", expected, st)
	}
}

func TestActionPlanCheckForASAP(t *testing.T) {
	at := &ActionPlan{Timing: &RateInterval{Timing: &RITiming{StartTime: utils.ASAP}}}
	if !at.IsASAP() {
		t.Errorf("%v should be asap!", at)
	}
}

func TestActionPlanLogFunction(t *testing.T) {
	a := &Action{
		ActionType:  "*log",
		BalanceType: "test",
		Balance:     &Balance{Value: 1.1},
	}
	at := &ActionPlan{
		actions: []*Action{a},
	}
	err := at.Execute()
	if err != nil {
		t.Errorf("Could not execute LOG action: %v", err)
	}
}

func TestActionPlanFunctionNotAvailable(t *testing.T) {
	a := &Action{
		ActionType:  "VALID_FUNCTION_TYPE",
		BalanceType: "test",
		Balance:     &Balance{Value: 1.1},
	}
	at := &ActionPlan{
		AccountIds: []string{"one", "two", "three"},
		Timing:     &RateInterval{},
		actions:    []*Action{a},
	}
	err := at.Execute()
	if at.Timing != nil {
		t.Errorf("Faild to detect wrong function type: %v", err)
	}
}

func TestActionPlanPriotityListSortByWeight(t *testing.T) {
	at1 := &ActionPlan{Timing: &RateInterval{
		Timing: &RITiming{
			Years:     utils.Years{2020},
			Months:    utils.Months{time.January, time.February, time.March, time.April, time.May, time.June, time.July, time.August, time.September, time.October, time.November, time.December},
			MonthDays: utils.MonthDays{1},
			StartTime: "00:00:00",
		},
		Weight: 20,
	}}
	at2 := &ActionPlan{Timing: &RateInterval{
		Timing: &RITiming{
			Years:     utils.Years{2020},
			Months:    utils.Months{time.January, time.February, time.March, time.April, time.May, time.June, time.July, time.August, time.September, time.October, time.November, time.December},
			MonthDays: utils.MonthDays{2},
			StartTime: "00:00:00",
		},
		Weight: 10,
	}}
	var atpl ActionPlanPriotityList
	atpl = append(atpl, at2, at1)
	atpl.Sort()
	if atpl[0] != at1 || atpl[1] != at2 {
		t.Error("Timing list not sorted correctly: ", at1, at2, atpl)
	}
}

func TestActionPlanPriotityListWeight(t *testing.T) {
	at1 := &ActionPlan{
		Timing: &RateInterval{
			Timing: &RITiming{
				Months:    utils.Months{time.January, time.February, time.March, time.April, time.May, time.June, time.July, time.August, time.September, time.October, time.November, time.December},
				MonthDays: utils.MonthDays{1},
				StartTime: "00:00:00",
			},
		},
		Weight: 20,
	}
	at2 := &ActionPlan{
		Timing: &RateInterval{
			Timing: &RITiming{
				Months:    utils.Months{time.January, time.February, time.March, time.April, time.May, time.June, time.July, time.August, time.September, time.October, time.November, time.December},
				MonthDays: utils.MonthDays{1},
				StartTime: "00:00:00",
			},
		},
		Weight: 10,
	}
	var atpl ActionPlanPriotityList
	atpl = append(atpl, at2, at1)
	atpl.Sort()
	if atpl[0] != at1 || atpl[1] != at2 {
		t.Error("Timing list not sorted correctly: ", atpl)
	}
}

func TestActionPlansRemoveMember(t *testing.T) {
	at1 := &ActionPlan{
		Uuid:       "some uuid",
		Id:         "test",
		AccountIds: []string{"one", "two", "three"},
		ActionsId:  "TEST_ACTIONS",
	}
	at2 := &ActionPlan{
		Uuid:       "some uuid22",
		Id:         "test2",
		AccountIds: []string{"three", "four"},
		ActionsId:  "TEST_ACTIONS2",
	}
	ats := ActionPlans{at1, at2}
	if outAts := RemActionPlan(ats, "", "four"); len(outAts[1].AccountIds) != 1 {
		t.Error("Expecting fewer balance ids", outAts[1].AccountIds)
	}
	if ats = RemActionPlan(ats, "", "three"); len(ats) != 1 {
		t.Error("Expecting fewer actionTimings", ats)
	}
	if ats = RemActionPlan(ats, "some_uuid22", ""); len(ats) != 1 {
		t.Error("Expecting fewer actionTimings members", ats)
	}
	ats2 := ActionPlans{at1, at2}
	if ats2 = RemActionPlan(ats2, "", ""); len(ats2) != 0 {
		t.Error("Should have no members anymore", ats2)
	}
}

func TestActionTriggerMatchNil(t *testing.T) {
	at := &ActionTrigger{
		BalanceDirection: OUTBOUND,
		BalanceType:      utils.MONETARY,
		ThresholdType:    TRIGGER_MAX_BALANCE,
		ThresholdValue:   2,
	}
	var a *Action
	if !at.Match(a) {
		t.Errorf("Action trigger [%v] does not match action [%v]", at, a)
	}
}

func TestActionTriggerMatchAllBlank(t *testing.T) {
	at := &ActionTrigger{
		BalanceDirection: OUTBOUND,
		BalanceType:      utils.MONETARY,
		ThresholdType:    TRIGGER_MAX_BALANCE,
		ThresholdValue:   2,
	}
	a := &Action{}
	if !at.Match(a) {
		t.Errorf("Action trigger [%v] does not match action [%v]", at, a)
	}
}

func TestActionTriggerMatchMinuteBucketBlank(t *testing.T) {
	at := &ActionTrigger{
		BalanceDirection: OUTBOUND,
		BalanceType:      utils.MONETARY,
		ThresholdType:    TRIGGER_MAX_BALANCE,
		ThresholdValue:   2,
	}
	a := &Action{Direction: OUTBOUND, BalanceType: utils.MONETARY}
	if !at.Match(a) {
		t.Errorf("Action trigger [%v] does not match action [%v]", at, a)
	}
}

func TestActionTriggerMatchMinuteBucketFull(t *testing.T) {
	at := &ActionTrigger{
		BalanceDirection: OUTBOUND,
		BalanceType:      utils.MONETARY,
		ThresholdType:    TRIGGER_MAX_BALANCE,
		ThresholdValue:   2,
	}
	a := &Action{ExtraParameters: fmt.Sprintf(`{"ThresholdType":"%v", "ThresholdValue": %v}`, TRIGGER_MAX_BALANCE, 2)}
	if !at.Match(a) {
		t.Errorf("Action trigger [%v] does not match action [%v]", at, a)
	}
}

func TestActionTriggerMatchAllFull(t *testing.T) {
	at := &ActionTrigger{
		BalanceDirection: OUTBOUND,
		BalanceType:      utils.MONETARY,
		ThresholdType:    TRIGGER_MAX_BALANCE,
		ThresholdValue:   2,
	}
	a := &Action{Direction: OUTBOUND, BalanceType: utils.MONETARY, ExtraParameters: fmt.Sprintf(`{"ThresholdType":"%v", "ThresholdValue": %v}`, TRIGGER_MAX_BALANCE, 2)}
	if !at.Match(a) {
		t.Errorf("Action trigger [%v] does not match action [%v]", at, a)
	}
}

func TestActionTriggerMatchSomeFalse(t *testing.T) {
	at := &ActionTrigger{
		BalanceDirection: OUTBOUND,
		BalanceType:      utils.MONETARY,
		ThresholdType:    TRIGGER_MAX_BALANCE,
		ThresholdValue:   2,
	}
	a := &Action{Direction: INBOUND, BalanceType: utils.MONETARY, ExtraParameters: fmt.Sprintf(`{"ThresholdType":"%v", "ThresholdValue": %v}`, TRIGGER_MAX_BALANCE, 2)}
	if at.Match(a) {
		t.Errorf("Action trigger [%v] does not match action [%v]", at, a)
	}
}

func TestActionTriggerMatcBalanceFalse(t *testing.T) {
	at := &ActionTrigger{
		BalanceDirection: OUTBOUND,
		BalanceType:      utils.MONETARY,
		ThresholdType:    TRIGGER_MAX_BALANCE,
		ThresholdValue:   2,
	}
	a := &Action{Direction: OUTBOUND, BalanceType: utils.MONETARY, ExtraParameters: fmt.Sprintf(`{"ThresholdType":"%v", "ThresholdValue": %v}`, TRIGGER_MAX_BALANCE, 3.0)}
	if at.Match(a) {
		t.Errorf("Action trigger [%v] does not match action [%v]", at, a)
	}
}

func TestActionTriggerMatcAllFalse(t *testing.T) {
	at := &ActionTrigger{
		BalanceDirection: OUTBOUND,
		BalanceType:      utils.MONETARY,
		ThresholdType:    TRIGGER_MAX_BALANCE,
		ThresholdValue:   2,
	}
	a := &Action{Direction: INBOUND, BalanceType: utils.VOICE, ExtraParameters: fmt.Sprintf(`{"ThresholdType":"%v", "ThresholdValue": %v}`, TRIGGER_MAX_COUNTER, 3)}
	if at.Match(a) {
		t.Errorf("Action trigger [%v] does not match action [%v]", at, a)
	}
}

func TestActionTriggerMatchAll(t *testing.T) {
	at := &ActionTrigger{
		BalanceDirection:      OUTBOUND,
		BalanceType:           utils.MONETARY,
		ThresholdType:         TRIGGER_MAX_BALANCE,
		ThresholdValue:        2,
		BalanceDestinationIds: "NAT",
		BalanceWeight:         1.0,
		BalanceRatingSubject:  "test1",
		BalanceSharedGroup:    "test2",
	}
	a := &Action{Direction: OUTBOUND, BalanceType: utils.MONETARY, ExtraParameters: fmt.Sprintf(`{"ThresholdType":"%v", "ThresholdValue": %v, "DestinationIds": "%v", "BalanceWeight": %v, "BalanceRatingSubject": "%v", "BalanceSharedGroup": "%v"}`, TRIGGER_MAX_BALANCE, 2, "NAT", 1.0, "test1", "test2")}
	if !at.Match(a) {
		t.Errorf("Action trigger [%v] does not match action [%v]", at, a)
	}
}

func TestActionTriggerPriotityList(t *testing.T) {
	at1 := &ActionTrigger{Weight: 30}
	at2 := &ActionTrigger{Weight: 20}
	at3 := &ActionTrigger{Weight: 10}
	var atpl ActionTriggerPriotityList
	atpl = append(atpl, at2, at1, at3)
	atpl.Sort()
	if atpl[0] != at1 || atpl[2] != at3 || atpl[1] != at2 {
		t.Error("List not sorted: ", atpl)
	}
}

func TestActionResetTriggres(t *testing.T) {
	ub := &Account{
		Id:             "TEST_UB",
		BalanceMap:     map[string]BalanceChain{utils.MONETARY: BalanceChain{&Balance{Value: 10}}, utils.VOICE + OUTBOUND: BalanceChain{&Balance{Value: 10, Weight: 20, DestinationIds: "NAT"}, &Balance{Weight: 10, DestinationIds: "RET"}}},
		UnitCounters:   []*UnitsCounter{&UnitsCounter{BalanceType: utils.MONETARY, Balances: BalanceChain{&Balance{Value: 1}}}},
		ActionTriggers: ActionTriggerPriotityList{&ActionTrigger{BalanceType: utils.MONETARY, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}, &ActionTrigger{BalanceType: utils.MONETARY, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}},
	}
	resetTriggersAction(ub, nil, nil, nil)
	if ub.ActionTriggers[0].Executed == true || ub.ActionTriggers[1].Executed == true {
		t.Error("Reset triggers action failed!")
	}
}

func TestActionResetTriggresExecutesThem(t *testing.T) {
	ub := &Account{
		Id:             "TEST_UB",
		BalanceMap:     map[string]BalanceChain{utils.MONETARY: BalanceChain{&Balance{Value: 10}}},
		UnitCounters:   []*UnitsCounter{&UnitsCounter{BalanceType: utils.MONETARY, Balances: BalanceChain{&Balance{Value: 1}}}},
		ActionTriggers: ActionTriggerPriotityList{&ActionTrigger{BalanceType: utils.MONETARY, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}},
	}
	resetTriggersAction(ub, nil, nil, nil)
	if ub.ActionTriggers[0].Executed == true || ub.BalanceMap[utils.MONETARY][0].GetValue() == 12 {
		t.Error("Reset triggers action failed!")
	}
}

func TestActionResetTriggresActionFilter(t *testing.T) {
	ub := &Account{
		Id:             "TEST_UB",
		BalanceMap:     map[string]BalanceChain{utils.MONETARY: BalanceChain{&Balance{Value: 10}}, utils.VOICE + OUTBOUND: BalanceChain{&Balance{Value: 10, Weight: 20, DestinationIds: "NAT"}, &Balance{Weight: 10, DestinationIds: "RET"}}},
		UnitCounters:   []*UnitsCounter{&UnitsCounter{BalanceType: utils.MONETARY, Balances: BalanceChain{&Balance{Value: 1}}}},
		ActionTriggers: ActionTriggerPriotityList{&ActionTrigger{BalanceType: utils.MONETARY, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}, &ActionTrigger{BalanceType: utils.MONETARY, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}},
	}
	resetTriggersAction(ub, nil, &Action{BalanceType: utils.SMS}, nil)
	if ub.ActionTriggers[0].Executed == false || ub.ActionTriggers[1].Executed == false {
		t.Error("Reset triggers action failed!")
	}
}

func TestActionSetPostpaid(t *testing.T) {
	ub := &Account{
		Id:             "TEST_UB",
		BalanceMap:     map[string]BalanceChain{utils.MONETARY: BalanceChain{&Balance{Value: 100}}, utils.VOICE + OUTBOUND: BalanceChain{&Balance{Value: 10, Weight: 20, DestinationIds: "NAT"}, &Balance{Weight: 10, DestinationIds: "RET"}}},
		UnitCounters:   []*UnitsCounter{&UnitsCounter{BalanceType: utils.MONETARY, Balances: BalanceChain{&Balance{Value: 1}}}},
		ActionTriggers: ActionTriggerPriotityList{&ActionTrigger{BalanceType: utils.MONETARY, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}, &ActionTrigger{BalanceType: utils.MONETARY, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}},
	}
	allowNegativeAction(ub, nil, nil, nil)
	if !ub.AllowNegative {
		t.Error("Set postpaid action failed!")
	}
}

func TestActionSetPrepaid(t *testing.T) {
	ub := &Account{
		Id:             "TEST_UB",
		AllowNegative:  true,
		BalanceMap:     map[string]BalanceChain{utils.MONETARY: BalanceChain{&Balance{Value: 100}}, utils.VOICE + OUTBOUND: BalanceChain{&Balance{Value: 10, Weight: 20, DestinationIds: "NAT"}, &Balance{Weight: 10, DestinationIds: "RET"}}},
		UnitCounters:   []*UnitsCounter{&UnitsCounter{BalanceType: utils.MONETARY, Balances: BalanceChain{&Balance{Value: 1}}}},
		ActionTriggers: ActionTriggerPriotityList{&ActionTrigger{BalanceType: utils.MONETARY, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}, &ActionTrigger{BalanceType: utils.MONETARY, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}},
	}
	denyNegativeAction(ub, nil, nil, nil)
	if ub.AllowNegative {
		t.Error("Set prepaid action failed!")
	}
}

func TestActionResetPrepaid(t *testing.T) {
	ub := &Account{
		Id:             "TEST_UB",
		AllowNegative:  true,
		BalanceMap:     map[string]BalanceChain{utils.MONETARY: BalanceChain{&Balance{Value: 100}}, utils.VOICE + OUTBOUND: BalanceChain{&Balance{Value: 10, Weight: 20, DestinationIds: "NAT"}, &Balance{Weight: 10, DestinationIds: "RET"}}},
		UnitCounters:   []*UnitsCounter{&UnitsCounter{BalanceType: utils.MONETARY, Balances: BalanceChain{&Balance{Value: 1}}}},
		ActionTriggers: ActionTriggerPriotityList{&ActionTrigger{BalanceType: utils.SMS, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}, &ActionTrigger{BalanceType: utils.SMS, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}},
	}
	resetAccountAction(ub, nil, nil, nil)
	if !ub.AllowNegative ||
		ub.BalanceMap[utils.MONETARY].GetTotalValue() != 0 ||
		len(ub.UnitCounters) != 0 ||
		ub.BalanceMap[utils.VOICE+OUTBOUND][0].GetValue() != 0 ||
		ub.ActionTriggers[0].Executed == true || ub.ActionTriggers[1].Executed == true {
		t.Log(ub.BalanceMap)
		t.Error("Reset prepaid action failed!")
	}
}

func TestActionResetPostpaid(t *testing.T) {
	ub := &Account{
		Id:             "TEST_UB",
		BalanceMap:     map[string]BalanceChain{utils.MONETARY: BalanceChain{&Balance{Value: 100}}, utils.VOICE + OUTBOUND: BalanceChain{&Balance{Value: 10, Weight: 20, DestinationIds: "NAT"}, &Balance{Weight: 10, DestinationIds: "RET"}}},
		UnitCounters:   []*UnitsCounter{&UnitsCounter{BalanceType: utils.MONETARY, Balances: BalanceChain{&Balance{Value: 1}}}},
		ActionTriggers: ActionTriggerPriotityList{&ActionTrigger{BalanceType: utils.SMS, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}, &ActionTrigger{BalanceType: utils.SMS, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}},
	}
	resetAccountAction(ub, nil, nil, nil)
	if ub.BalanceMap[utils.MONETARY].GetTotalValue() != 0 ||
		len(ub.UnitCounters) != 0 ||
		ub.BalanceMap[utils.VOICE+OUTBOUND][0].GetValue() != 0 ||
		ub.ActionTriggers[0].Executed == true || ub.ActionTriggers[1].Executed == true {
		t.Error("Reset postpaid action failed!")
	}
}

func TestActionTopupResetCredit(t *testing.T) {
	ub := &Account{
		Id:             "TEST_UB",
		BalanceMap:     map[string]BalanceChain{utils.MONETARY + OUTBOUND: BalanceChain{&Balance{Value: 100}}, utils.VOICE + OUTBOUND: BalanceChain{&Balance{Value: 10, Weight: 20, DestinationIds: "NAT"}, &Balance{Weight: 10, DestinationIds: "RET"}}},
		UnitCounters:   []*UnitsCounter{&UnitsCounter{BalanceType: utils.MONETARY, Direction: OUTBOUND, Balances: BalanceChain{&Balance{Value: 1}}}},
		ActionTriggers: ActionTriggerPriotityList{&ActionTrigger{BalanceType: utils.MONETARY, BalanceDirection: OUTBOUND, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}, &ActionTrigger{BalanceType: utils.MONETARY, BalanceDirection: OUTBOUND, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}},
	}
	a := &Action{BalanceType: utils.MONETARY, Direction: OUTBOUND, Balance: &Balance{Value: 10}}
	topupResetAction(ub, nil, a, nil)
	if ub.AllowNegative ||
		ub.BalanceMap[utils.MONETARY+OUTBOUND].GetTotalValue() != 10 ||
		len(ub.UnitCounters) != 1 ||
		len(ub.BalanceMap[utils.VOICE+OUTBOUND]) != 2 ||
		ub.ActionTriggers[0].Executed != true || ub.ActionTriggers[1].Executed != true {
		t.Errorf("Topup reset action failed: %+v", ub.BalanceMap[utils.MONETARY+OUTBOUND][0])
	}
}

func TestActionTopupResetCreditId(t *testing.T) {
	ub := &Account{
		Id: "TEST_UB",
		BalanceMap: map[string]BalanceChain{
			utils.MONETARY + OUTBOUND: BalanceChain{
				&Balance{Value: 100},
				&Balance{Id: "TEST_B", Value: 15},
			},
		},
	}
	a := &Action{BalanceType: utils.MONETARY, Direction: OUTBOUND, Balance: &Balance{Id: "TEST_B", Value: 10}}
	topupResetAction(ub, nil, a, nil)
	if ub.AllowNegative ||
		ub.BalanceMap[utils.MONETARY+OUTBOUND].GetTotalValue() != 110 ||
		len(ub.BalanceMap[utils.MONETARY+OUTBOUND]) != 2 {
		t.Errorf("Topup reset action failed: %+v", ub.BalanceMap[utils.MONETARY+OUTBOUND][0])
	}
}

func TestActionTopupResetCreditNoId(t *testing.T) {
	ub := &Account{
		Id: "TEST_UB",
		BalanceMap: map[string]BalanceChain{
			utils.MONETARY + OUTBOUND: BalanceChain{
				&Balance{Value: 100},
				&Balance{Id: "TEST_B", Value: 15},
			},
		},
	}
	a := &Action{BalanceType: utils.MONETARY, Direction: OUTBOUND, Balance: &Balance{Value: 10}}
	topupResetAction(ub, nil, a, nil)
	if ub.AllowNegative ||
		ub.BalanceMap[utils.MONETARY+OUTBOUND].GetTotalValue() != 20 ||
		len(ub.BalanceMap[utils.MONETARY+OUTBOUND]) != 2 {
		t.Errorf("Topup reset action failed: %+v", ub.BalanceMap[utils.MONETARY+OUTBOUND][1])
	}
}

func TestActionTopupResetMinutes(t *testing.T) {
	ub := &Account{
		Id: "TEST_UB",
		BalanceMap: map[string]BalanceChain{
			utils.MONETARY + OUTBOUND: BalanceChain{&Balance{Value: 100}},
			utils.VOICE + OUTBOUND:    BalanceChain{&Balance{Value: 10, Weight: 20, DestinationIds: "NAT"}, &Balance{Weight: 10, DestinationIds: "RET"}}},
		UnitCounters:   []*UnitsCounter{&UnitsCounter{BalanceType: utils.MONETARY, Direction: OUTBOUND, Balances: BalanceChain{&Balance{Value: 1}}}},
		ActionTriggers: ActionTriggerPriotityList{&ActionTrigger{BalanceType: utils.MONETARY, BalanceDirection: OUTBOUND, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}, &ActionTrigger{BalanceType: utils.MONETARY, BalanceDirection: OUTBOUND, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}},
	}
	a := &Action{BalanceType: utils.VOICE, Direction: OUTBOUND, Balance: &Balance{Value: 5, Weight: 20, DestinationIds: "NAT"}}
	topupResetAction(ub, nil, a, nil)
	if ub.AllowNegative ||
		ub.BalanceMap[utils.VOICE+OUTBOUND].GetTotalValue() != 5 ||
		ub.BalanceMap[utils.MONETARY+OUTBOUND].GetTotalValue() != 100 ||
		len(ub.UnitCounters) != 1 ||
		len(ub.BalanceMap[utils.VOICE+OUTBOUND]) != 2 ||
		ub.ActionTriggers[0].Executed != true || ub.ActionTriggers[1].Executed != true {
		t.Errorf("Topup reset minutes action failed: %+v", ub.BalanceMap[utils.VOICE+OUTBOUND][0])
	}
}

func TestActionTopupCredit(t *testing.T) {
	ub := &Account{
		Id:             "TEST_UB",
		BalanceMap:     map[string]BalanceChain{utils.MONETARY + OUTBOUND: BalanceChain{&Balance{Value: 100}}, utils.VOICE + OUTBOUND: BalanceChain{&Balance{Value: 10, Weight: 20, DestinationIds: "NAT"}, &Balance{Weight: 10, DestinationIds: "RET"}}},
		UnitCounters:   []*UnitsCounter{&UnitsCounter{BalanceType: utils.MONETARY, Direction: OUTBOUND, Balances: BalanceChain{&Balance{Value: 1}}}},
		ActionTriggers: ActionTriggerPriotityList{&ActionTrigger{BalanceType: utils.MONETARY, BalanceDirection: OUTBOUND, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}, &ActionTrigger{BalanceType: utils.MONETARY, BalanceDirection: OUTBOUND, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}},
	}
	a := &Action{BalanceType: utils.MONETARY, Direction: OUTBOUND, Balance: &Balance{Value: 10}}
	topupAction(ub, nil, a, nil)
	if ub.AllowNegative ||
		ub.BalanceMap[utils.MONETARY+OUTBOUND].GetTotalValue() != 110 ||
		len(ub.UnitCounters) != 1 ||
		len(ub.BalanceMap[utils.VOICE+OUTBOUND]) != 2 ||
		ub.ActionTriggers[0].Executed != true || ub.ActionTriggers[1].Executed != true {
		t.Error("Topup action failed!", ub.BalanceMap[utils.MONETARY+OUTBOUND].GetTotalValue())
	}
}

func TestActionTopupMinutes(t *testing.T) {
	ub := &Account{
		Id:             "TEST_UB",
		BalanceMap:     map[string]BalanceChain{utils.MONETARY: BalanceChain{&Balance{Value: 100}}, utils.VOICE + OUTBOUND: BalanceChain{&Balance{Value: 10, Weight: 20, DestinationIds: "NAT"}, &Balance{Weight: 10, DestinationIds: "RET"}}},
		UnitCounters:   []*UnitsCounter{&UnitsCounter{BalanceType: utils.MONETARY, Balances: BalanceChain{&Balance{Value: 1}}}},
		ActionTriggers: ActionTriggerPriotityList{&ActionTrigger{BalanceType: utils.MONETARY, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}, &ActionTrigger{BalanceType: utils.MONETARY, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}},
	}
	a := &Action{BalanceType: utils.VOICE, Direction: OUTBOUND, Balance: &Balance{Value: 5, Weight: 20, DestinationIds: "NAT"}}
	topupAction(ub, nil, a, nil)
	if ub.AllowNegative ||
		ub.BalanceMap[utils.VOICE+OUTBOUND].GetTotalValue() != 15 ||
		ub.BalanceMap[utils.MONETARY].GetTotalValue() != 100 ||
		len(ub.UnitCounters) != 1 ||
		len(ub.BalanceMap[utils.VOICE+OUTBOUND]) != 2 ||
		ub.ActionTriggers[0].Executed != true || ub.ActionTriggers[1].Executed != true {
		t.Error("Topup minutes action failed!", ub.BalanceMap[utils.VOICE+OUTBOUND])
	}
}

func TestActionDebitCredit(t *testing.T) {
	ub := &Account{
		Id:             "TEST_UB",
		BalanceMap:     map[string]BalanceChain{utils.MONETARY + OUTBOUND: BalanceChain{&Balance{Value: 100}}, utils.VOICE + OUTBOUND: BalanceChain{&Balance{Value: 10, Weight: 20, DestinationIds: "NAT"}, &Balance{Weight: 10, DestinationIds: "RET"}}},
		UnitCounters:   []*UnitsCounter{&UnitsCounter{BalanceType: utils.MONETARY, Direction: OUTBOUND, Balances: BalanceChain{&Balance{Value: 1}}}},
		ActionTriggers: ActionTriggerPriotityList{&ActionTrigger{BalanceType: utils.MONETARY, BalanceDirection: OUTBOUND, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}, &ActionTrigger{BalanceType: utils.MONETARY, BalanceDirection: OUTBOUND, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}},
	}
	a := &Action{BalanceType: utils.MONETARY, Direction: OUTBOUND, Balance: &Balance{Value: 10}}
	debitAction(ub, nil, a, nil)
	if ub.AllowNegative ||
		ub.BalanceMap[utils.MONETARY+OUTBOUND].GetTotalValue() != 90 ||
		len(ub.UnitCounters) != 1 ||
		len(ub.BalanceMap[utils.VOICE+OUTBOUND]) != 2 ||
		ub.ActionTriggers[0].Executed != true || ub.ActionTriggers[1].Executed != true {
		t.Error("Debit action failed!", ub.BalanceMap[utils.MONETARY+OUTBOUND].GetTotalValue())
	}
}

func TestActionDebitMinutes(t *testing.T) {
	ub := &Account{
		Id:             "TEST_UB",
		BalanceMap:     map[string]BalanceChain{utils.MONETARY: BalanceChain{&Balance{Value: 100}}, utils.VOICE + OUTBOUND: BalanceChain{&Balance{Value: 10, Weight: 20, DestinationIds: "NAT"}, &Balance{Weight: 10, DestinationIds: "RET"}}},
		UnitCounters:   []*UnitsCounter{&UnitsCounter{BalanceType: utils.MONETARY, Balances: BalanceChain{&Balance{Value: 1}}}},
		ActionTriggers: ActionTriggerPriotityList{&ActionTrigger{BalanceType: utils.MONETARY, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}, &ActionTrigger{BalanceType: utils.MONETARY, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}},
	}
	a := &Action{BalanceType: utils.VOICE, Direction: OUTBOUND, Balance: &Balance{Value: 5, Weight: 20, DestinationIds: "NAT"}}
	debitAction(ub, nil, a, nil)
	if ub.AllowNegative ||
		ub.BalanceMap[utils.VOICE+OUTBOUND][0].GetValue() != 5 ||
		ub.BalanceMap[utils.MONETARY].GetTotalValue() != 100 ||
		len(ub.UnitCounters) != 1 ||
		len(ub.BalanceMap[utils.VOICE+OUTBOUND]) != 2 ||
		ub.ActionTriggers[0].Executed != true || ub.ActionTriggers[1].Executed != true {
		t.Error("Debit minutes action failed!", ub.BalanceMap[utils.VOICE+OUTBOUND][0])
	}
}

func TestActionResetAllCounters(t *testing.T) {
	ub := &Account{
		Id:            "TEST_UB",
		AllowNegative: true,
		BalanceMap: map[string]BalanceChain{
			utils.MONETARY: BalanceChain{&Balance{Value: 100}},
			utils.VOICE: BalanceChain{
				&Balance{Value: 10, Weight: 20, DestinationIds: "NAT"},
				&Balance{Weight: 10, DestinationIds: "RET"}}},
		UnitCounters:   []*UnitsCounter{&UnitsCounter{BalanceType: utils.MONETARY, Balances: BalanceChain{&Balance{Value: 1}}}},
		ActionTriggers: ActionTriggerPriotityList{&ActionTrigger{BalanceType: utils.MONETARY, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}},
	}
	resetCountersAction(ub, nil, nil, nil)
	if !ub.AllowNegative ||
		ub.BalanceMap[utils.MONETARY].GetTotalValue() != 100 ||
		len(ub.UnitCounters) != 1 ||
		len(ub.UnitCounters[0].Balances) != 2 ||
		len(ub.BalanceMap[utils.VOICE]) != 2 ||
		ub.ActionTriggers[0].Executed != true {
		t.Errorf("Reset counters action failed: %+v", ub.UnitCounters)
	}
	if len(ub.UnitCounters) < 1 {
		t.FailNow()
	}
	mb := ub.UnitCounters[0].Balances[0]
	if mb.Weight != 20 || mb.GetValue() != 0 || mb.DestinationIds != "NAT" {
		t.Errorf("Balance cloned incorrectly: %v!", mb)
	}
}

func TestActionResetCounterMinutes(t *testing.T) {
	ub := &Account{
		Id:            "TEST_UB",
		AllowNegative: true,
		BalanceMap: map[string]BalanceChain{
			utils.MONETARY: BalanceChain{&Balance{Value: 100}},
			utils.VOICE:    BalanceChain{&Balance{Value: 10, Weight: 20, DestinationIds: "NAT"}, &Balance{Weight: 10, DestinationIds: "RET"}}},
		UnitCounters:   []*UnitsCounter{&UnitsCounter{BalanceType: utils.MONETARY, Balances: BalanceChain{&Balance{Value: 1}}}},
		ActionTriggers: ActionTriggerPriotityList{&ActionTrigger{BalanceType: utils.MONETARY, ThresholdType: "*max_counter", ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}},
	}
	a := &Action{BalanceType: utils.VOICE}
	resetCounterAction(ub, nil, a, nil)
	if !ub.AllowNegative ||
		ub.BalanceMap[utils.MONETARY].GetTotalValue() != 100 ||
		len(ub.UnitCounters) != 2 ||
		len(ub.UnitCounters[1].Balances) != 2 ||
		len(ub.BalanceMap[utils.VOICE]) != 2 ||
		ub.ActionTriggers[0].Executed != true {
		for _, b := range ub.UnitCounters[1].Balances {
			t.Logf("B: %+v", b)
		}
		t.Errorf("Reset counters action failed: %+v", ub)
	}
	if len(ub.UnitCounters) < 2 || len(ub.UnitCounters[1].Balances) < 1 {
		t.FailNow()
	}
	mb := ub.UnitCounters[1].Balances[0]
	if mb.Weight != 20 || mb.GetValue() != 0 || mb.DestinationIds != "NAT" {
		t.Errorf("Balance cloned incorrectly: %+v!", mb)
	}
}

func TestActionResetCounterCredit(t *testing.T) {
	ub := &Account{
		Id:             "TEST_UB",
		AllowNegative:  true,
		BalanceMap:     map[string]BalanceChain{utils.MONETARY: BalanceChain{&Balance{Value: 100}}, utils.VOICE + OUTBOUND: BalanceChain{&Balance{Value: 10, Weight: 20, DestinationIds: "NAT"}, &Balance{Weight: 10, DestinationIds: "RET"}}},
		UnitCounters:   []*UnitsCounter{&UnitsCounter{BalanceType: utils.MONETARY, Direction: OUTBOUND, Balances: BalanceChain{&Balance{Value: 1}}}, &UnitsCounter{BalanceType: utils.SMS, Direction: OUTBOUND, Balances: BalanceChain{&Balance{Value: 1}}}},
		ActionTriggers: ActionTriggerPriotityList{&ActionTrigger{BalanceType: utils.MONETARY, BalanceDirection: OUTBOUND, ThresholdValue: 2, ActionsId: "TEST_ACTIONS", Executed: true}},
	}
	a := &Action{BalanceType: utils.MONETARY, Direction: OUTBOUND}
	resetCounterAction(ub, nil, a, nil)
	if !ub.AllowNegative ||
		ub.BalanceMap[utils.MONETARY].GetTotalValue() != 100 ||
		len(ub.UnitCounters) != 2 ||
		len(ub.BalanceMap[utils.VOICE+OUTBOUND]) != 2 ||
		ub.ActionTriggers[0].Executed != true {
		t.Error("Reset counters action failed!", ub.UnitCounters)
	}
}

func TestActionTriggerLogging(t *testing.T) {
	at := &ActionTrigger{
		Id:                    "some_uuid",
		BalanceType:           utils.MONETARY,
		BalanceDirection:      OUTBOUND,
		ThresholdValue:        100.0,
		BalanceDestinationIds: "NAT",
		Weight:                10.0,
		ActionsId:             "TEST_ACTIONS",
	}
	as, err := ratingStorage.GetActions(at.ActionsId, false)
	if err != nil {
		t.Error("Error getting actions for the action timing: ", as, err)
	}
	storageLogger.LogActionTrigger("rif", utils.RATER_SOURCE, at, as)
	//expected := "rif*some_uuid;MONETARY;OUT;NAT;TEST_ACTIONS;100;10;false*|TOPUP|MONETARY|OUT|10|0"
	var key string
	atMap, _ := ratingStorage.GetAllActionPlans()
	for k, v := range atMap {
		_ = k
		_ = v
		/*if strings.Contains(k, LOG_ACTION_TRIGGER_PREFIX) && strings.Contains(v, expected) {
		    key = k
		    break
		}*/
	}
	if key != "" {
		t.Error("Action timing was not logged")
	}
}

func TestActionPlanLogging(t *testing.T) {
	i := &RateInterval{
		Timing: &RITiming{
			Months:    utils.Months{time.January, time.February, time.March, time.April, time.May, time.June, time.July, time.August, time.September, time.October, time.November, time.December},
			MonthDays: utils.MonthDays{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			WeekDays:  utils.WeekDays{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
			StartTime: "18:00:00",
			EndTime:   "00:00:00",
		},
		Weight: 10.0,
		Rating: &RIRate{
			ConnectFee: 0.0,
			Rates:      RateGroups{&Rate{0, 1.0, 1 * time.Second, 60 * time.Second}},
		},
	}
	at := &ActionPlan{
		Uuid:       "some uuid",
		Id:         "test",
		AccountIds: []string{"one", "two", "three"},
		Timing:     i,
		Weight:     10.0,
		ActionsId:  "TEST_ACTIONS",
	}
	as, err := ratingStorage.GetActions(at.ActionsId, false)
	if err != nil {
		t.Error("Error getting actions for the action trigger: ", err)
	}
	storageLogger.LogActionPlan(utils.SCHED_SOURCE, at, as)
	//expected := "some uuid|test|one,two,three|;1,2,3,4,5,6,7,8,9,10,11,12;1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31;1,2,3,4,5;18:00:00;00:00:00;10;0;1;60;1|10|TEST_ACTIONS*|TOPUP|MONETARY|OUT|10|0"
	var key string
	atMap, _ := ratingStorage.GetAllActionPlans()
	for k, v := range atMap {
		_ = k
		_ = v
		/*if strings.Contains(k, LOG_ACTION_TIMMING_PREFIX) && strings.Contains(string(v), expected) {
		    key = k
		}*/
	}
	if key != "" {
		t.Error("Action trigger was not logged")
	}
}

func TestActionMakeNegative(t *testing.T) {
	a := &Action{Balance: &Balance{Value: 10}}
	genericMakeNegative(a)
	if a.Balance.GetValue() > 0 {
		t.Error("Failed to make negative: ", a)
	}
	genericMakeNegative(a)
	if a.Balance.GetValue() > 0 {
		t.Error("Failed to preserve negative: ", a)
	}
}

func TestRemoveAction(t *testing.T) {
	if _, err := accountingStorage.GetAccount("*out:cgrates.org:remo"); err != nil {
		t.Errorf("account to be removed not found: %v", err)
	}
	a := &Action{
		ActionType: REMOVE_ACCOUNT,
	}

	at := &ActionPlan{
		AccountIds: []string{"*out:cgrates.org:remo"},
		actions:    Actions{a},
	}

	at.Execute()
	afterUb, err := accountingStorage.GetAccount("*out:cgrates.org:remo")
	if err != utils.ErrNotFound || afterUb != nil {
		t.Error("error removing account: ", err)
	}
}

func TestTopupAction(t *testing.T) {
	initialUb, _ := accountingStorage.GetAccount("*out:vdf:minu")
	a := &Action{
		ActionType:  TOPUP,
		BalanceType: utils.MONETARY,
		Direction:   OUTBOUND,
		Balance:     &Balance{Value: 25, DestinationIds: "RET", Weight: 20},
	}

	at := &ActionPlan{
		AccountIds: []string{"*out:vdf:minu"},
		actions:    Actions{a},
	}

	at.Execute()
	afterUb, _ := accountingStorage.GetAccount("*out:vdf:minu")
	initialValue := initialUb.BalanceMap[utils.MONETARY+OUTBOUND].GetTotalValue()
	afterValue := afterUb.BalanceMap[utils.MONETARY+OUTBOUND].GetTotalValue()
	if initialValue != 50 || afterValue != 75 {
		t.Error("Bad topup before and after: ", initialValue, afterValue)
	}
}

func TestTopupActionLoaded(t *testing.T) {
	initialUb, _ := accountingStorage.GetAccount("*out:vdf:minitsboy")
	a := &Action{
		ActionType:  TOPUP,
		BalanceType: utils.MONETARY,
		Direction:   OUTBOUND,
		Balance:     &Balance{Value: 25, DestinationIds: "RET", Weight: 20},
	}

	at := &ActionPlan{
		AccountIds: []string{"*out:vdf:minitsboy"},
		actions:    Actions{a},
	}

	at.Execute()
	afterUb, _ := accountingStorage.GetAccount("*out:vdf:minitsboy")
	initialValue := initialUb.BalanceMap[utils.MONETARY+OUTBOUND].GetTotalValue()
	afterValue := afterUb.BalanceMap[utils.MONETARY+OUTBOUND].GetTotalValue()
	if initialValue != 100 || afterValue != 125 {
		t.Logf("Initial: %+v", initialUb)
		t.Logf("After: %+v", afterUb)
		t.Error("Bad topup before and after: ", initialValue, afterValue)
	}
}

func TestActionCdrlogEmpty(t *testing.T) {
	acnt := &Account{Id: "*out:cgrates.org:dan2904"}
	cdrlog := &Action{
		ActionType: CDRLOG,
	}
	err := cdrLogAction(acnt, nil, cdrlog, Actions{
		&Action{
			ActionType: DEBIT,
			Balance:    &Balance{Value: 25, DestinationIds: "RET", Weight: 20},
		},
	})
	if err != nil {
		t.Error("Error performing cdrlog action: ", err)
	}
	cdrs := make([]*StoredCdr, 0)
	json.Unmarshal([]byte(cdrlog.ExpirationString), &cdrs)
	if len(cdrs) != 1 || cdrs[0].CdrSource != CDRLOG {
		t.Errorf("Wrong cdrlogs: %+v", cdrs[0])
	}
}

func TestActionCdrlogWithParams(t *testing.T) {
	acnt := &Account{Id: "*out:cgrates.org:dan2904"}
	cdrlog := &Action{
		ActionType:      CDRLOG,
		ExtraParameters: `{"ReqType":"^*pseudoprepaid","Subject":"^rif", "TOR":"~action_type:s/^\\*(.*)$/did_$1/"}`,
	}
	err := cdrLogAction(acnt, nil, cdrlog, Actions{
		&Action{
			ActionType: DEBIT,
			Balance:    &Balance{Value: 25, DestinationIds: "RET", Weight: 20},
		},
		&Action{
			ActionType: DEBIT_RESET,
			Balance:    &Balance{Value: 25, DestinationIds: "RET", Weight: 20},
		},
	})
	if err != nil {
		t.Error("Error performing cdrlog action: ", err)
	}
	cdrs := make([]*StoredCdr, 0)
	json.Unmarshal([]byte(cdrlog.ExpirationString), &cdrs)
	if len(cdrs) != 2 ||
		cdrs[0].Subject != "rif" {
		t.Errorf("Wrong cdrlogs: %+v", cdrs[0])
	}
}

func TestActionCdrLogParamsWithOverload(t *testing.T) {
	acnt := &Account{Id: "*out:cgrates.org:dan2904"}
	cdrlog := &Action{
		ActionType:      CDRLOG,
		ExtraParameters: `{"Subject":"^rif","Destination":"^1234","TOR":"~action_tag:s/^at(.)$/0$1/","AccountId":"~account_id:s/^\\*(.*)$/$1/"}`,
	}
	err := cdrLogAction(acnt, nil, cdrlog, Actions{
		&Action{
			ActionType: DEBIT,
			Balance:    &Balance{Value: 25, DestinationIds: "RET", Weight: 20},
		},
		&Action{
			ActionType: DEBIT_RESET,
			Balance:    &Balance{Value: 25, DestinationIds: "RET", Weight: 20},
		},
	})
	if err != nil {
		t.Error("Error performing cdrlog action: ", err)
	}
	cdrs := make([]*StoredCdr, 0)
	json.Unmarshal([]byte(cdrlog.ExpirationString), &cdrs)
	expectedExtraFields := map[string]string{
		"AccountId": "out:cgrates.org:dan2904",
	}
	if len(cdrs) != 2 ||
		cdrs[0].Subject != "rif" {
		t.Errorf("Wrong cdrlogs: %+v", cdrs[0])
	}
	if !reflect.DeepEqual(cdrs[0].ExtraFields, expectedExtraFields) {
		t.Errorf("Received extra_fields: %+v", cdrs[0].ExtraFields)
	}
}

/********************************** Benchmarks ********************************/

func BenchmarkUUID(b *testing.B) {
	m := make(map[string]int, 1000)
	for i := 0; i < b.N; i++ {
		uuid := utils.GenUUID()
		if len(uuid) == 0 {
			b.Fatalf("GenUUID error %s", uuid)
		}
		b.StopTimer()
		c := m[uuid]
		if c > 0 {
			b.Fatalf("duplicate uuid[%s] count %d", uuid, c)
		}
		m[uuid] = c + 1
		b.StartTimer()
	}
}
