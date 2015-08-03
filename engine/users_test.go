package engine

import (
	"reflect"
	"testing"
	"time"

	"github.com/cgrates/cgrates/utils"
)

var testMap = UserMap{
	table: map[string]map[string]string{
		"test:user":   map[string]string{"t": "v"},
		":user":       map[string]string{"t": "v"},
		"test:":       map[string]string{"t": "v"},
		"test1:user1": map[string]string{"t": "v", "x": "y"},
	},
	index: make(map[string]map[string]bool),
}

func TestUsersAdd(t *testing.T) {
	tm := newUserMap(accountingStorage, nil)
	var r string
	up := UserProfile{
		Tenant:   "test",
		UserName: "user",
		Profile: map[string]string{
			"t": "v",
		},
	}
	tm.SetUser(up, &r)
	p, found := tm.table[up.GetId()]
	if r != utils.OK ||
		!found ||
		p["t"] != "v" ||
		len(tm.table) != 1 ||
		len(p) != 1 {
		t.Error("Error setting user: ", tm, len(tm.table))
	}
}

func TestUsersUpdate(t *testing.T) {
	tm := newUserMap(accountingStorage, nil)
	var r string
	up := UserProfile{
		Tenant:   "test",
		UserName: "user",
		Profile: map[string]string{
			"t": "v",
		},
	}
	tm.SetUser(up, &r)
	p, found := tm.table[up.GetId()]
	if r != utils.OK ||
		!found ||
		p["t"] != "v" ||
		len(tm.table) != 1 ||
		len(p) != 1 {
		t.Error("Error setting user: ", tm)
	}
	up.Profile["x"] = "y"
	tm.UpdateUser(up, &r)
	p, found = tm.table[up.GetId()]
	if r != utils.OK ||
		!found ||
		p["x"] != "y" ||
		len(tm.table) != 1 ||
		len(p) != 2 {
		t.Error("Error updating user: ", tm)
	}
}

func TestUsersUpdateNotFound(t *testing.T) {
	tm := newUserMap(accountingStorage, nil)
	var r string
	up := UserProfile{
		Tenant:   "test",
		UserName: "user",
		Profile: map[string]string{
			"t": "v",
		},
	}
	tm.SetUser(up, &r)
	up.UserName = "test1"
	err = tm.UpdateUser(up, &r)
	if err != utils.ErrNotFound {
		t.Error("Error detecting user not found on update: ", err)
	}
}

func TestUsersUpdateInit(t *testing.T) {
	tm := newUserMap(accountingStorage, nil)
	var r string
	up := UserProfile{
		Tenant:   "test",
		UserName: "user",
	}
	tm.SetUser(up, &r)
	up = UserProfile{
		Tenant:   "test",
		UserName: "user",
		Profile: map[string]string{
			"t": "v",
		},
	}
	tm.UpdateUser(up, &r)
	p, found := tm.table[up.GetId()]
	if r != utils.OK ||
		!found ||
		p["t"] != "v" ||
		len(tm.table) != 1 ||
		len(p) != 1 {
		t.Error("Error updating user: ", tm)
	}
}

func TestUsersRemove(t *testing.T) {
	tm := newUserMap(accountingStorage, nil)
	var r string
	up := UserProfile{
		Tenant:   "test",
		UserName: "user",
		Profile: map[string]string{
			"t": "v",
		},
	}
	tm.SetUser(up, &r)
	p, found := tm.table[up.GetId()]
	if r != utils.OK ||
		!found ||
		p["t"] != "v" ||
		len(tm.table) != 1 ||
		len(p) != 1 {
		t.Error("Error setting user: ", tm)
	}
	tm.RemoveUser(up, &r)
	p, found = tm.table[up.GetId()]
	if r != utils.OK ||
		found ||
		len(tm.table) != 0 {
		t.Error("Error removing user: ", tm)
	}
}

func TestUsersGetFull(t *testing.T) {
	up := UserProfile{
		Tenant:   "test",
		UserName: "user",
		Profile: map[string]string{
			"t": "v",
		},
	}
	results := UserProfiles{}
	testMap.GetUsers(up, &results)
	if len(results) != 3 {
		t.Error("error getting users: ", results)
	}
}

func TestUsersGetTenant(t *testing.T) {
	up := UserProfile{
		Tenant:   "testX",
		UserName: "user",
		Profile: map[string]string{
			"t": "v",
		},
	}
	results := UserProfiles{}
	testMap.GetUsers(up, &results)
	if len(results) != 1 {
		t.Error("error getting users: ", results)
	}
}

func TestUsersGetUserName(t *testing.T) {
	up := UserProfile{
		Tenant:   "test",
		UserName: "userX",
		Profile: map[string]string{
			"t": "v",
		},
	}
	results := UserProfiles{}
	testMap.GetUsers(up, &results)
	if len(results) != 1 {
		t.Error("error getting users: ", results)
	}
}

func TestUsersGetNotFoundProfile(t *testing.T) {
	up := UserProfile{
		Tenant:   "test",
		UserName: "user",
		Profile: map[string]string{
			"o": "p",
		},
	}
	results := UserProfiles{}
	testMap.GetUsers(up, &results)
	if len(results) != 3 {
		t.Error("error getting users: ", results)
	}
}

func TestUsersGetMissingTenant(t *testing.T) {
	up := UserProfile{
		UserName: "user",
		Profile: map[string]string{
			"t": "v",
		},
	}
	results := UserProfiles{}
	testMap.GetUsers(up, &results)
	if len(results) != 3 {
		t.Error("error getting users: ", results)
	}
}

func TestUsersGetMissingUserName(t *testing.T) {
	up := UserProfile{
		Tenant: "test",
		Profile: map[string]string{
			"t": "v",
		},
	}
	results := UserProfiles{}
	testMap.GetUsers(up, &results)
	if len(results) != 3 {
		t.Error("error getting users: ", results)
	}
}

func TestUsersGetMissingId(t *testing.T) {
	up := UserProfile{
		Profile: map[string]string{
			"t": "v",
		},
	}
	results := UserProfiles{}
	testMap.GetUsers(up, &results)
	if len(results) != 4 {
		t.Error("error getting users: ", results)
	}
}

func TestUsersGetMissingIdTwo(t *testing.T) {
	up := UserProfile{
		Profile: map[string]string{
			"t": "v",
			"x": "y",
		},
	}
	results := UserProfiles{}
	testMap.GetUsers(up, &results)
	if len(results) != 4 {
		t.Error("error getting users: ", results)
	}
}

func TestUsersGetMissingIdTwoSort(t *testing.T) {
	up := UserProfile{
		Profile: map[string]string{
			"t": "v",
			"x": "y",
		},
	}
	results := UserProfiles{}
	testMap.GetUsers(up, &results)
	if len(results) != 4 {
		t.Error("error getting users: ", results)
	}
	if results[0].GetId() != "test1:user1" {
		t.Errorf("Error sorting profiles: %+v", results[0])
	}
}

func TestUsersAddIndex(t *testing.T) {
	var r string
	testMap.AddIndex([]string{"t"}, &r)
	if r != utils.OK ||
		len(testMap.index) != 1 ||
		len(testMap.index[utils.ConcatenatedKey("t", "v")]) != 4 {
		t.Error("error adding index: ", testMap.index)
	}
}

func TestUsersAddIndexFull(t *testing.T) {
	var r string
	testMap.index = make(map[string]map[string]bool) // reset index
	testMap.AddIndex([]string{"t", "x", "UserName", "Tenant"}, &r)
	if r != utils.OK ||
		len(testMap.index) != 6 ||
		len(testMap.index[utils.ConcatenatedKey("t", "v")]) != 4 {
		t.Error("error adding index: ", testMap.index)
	}
}

func TestUsersAddIndexNone(t *testing.T) {
	var r string
	testMap.index = make(map[string]map[string]bool) // reset index
	testMap.AddIndex([]string{"test"}, &r)
	if r != utils.OK ||
		len(testMap.index) != 0 {
		t.Error("error adding index: ", testMap.index)
	}
}

func TestUsersGetFullindex(t *testing.T) {
	var r string
	testMap.index = make(map[string]map[string]bool) // reset index
	testMap.AddIndex([]string{"t", "x", "UserName", "Tenant"}, &r)
	up := UserProfile{
		Tenant:   "test",
		UserName: "user",
		Profile: map[string]string{
			"t": "v",
		},
	}
	results := UserProfiles{}
	testMap.GetUsers(up, &results)
	if len(results) != 3 {
		t.Error("error getting users: ", results)
	}
}

func TestUsersGetTenantindex(t *testing.T) {
	var r string
	testMap.index = make(map[string]map[string]bool) // reset index
	testMap.AddIndex([]string{"t", "x", "UserName", "Tenant"}, &r)
	up := UserProfile{
		Tenant:   "testX",
		UserName: "user",
		Profile: map[string]string{
			"t": "v",
		},
	}
	results := UserProfiles{}
	testMap.GetUsers(up, &results)
	if len(results) != 1 {
		t.Error("error getting users: ", results)
	}
}

func TestUsersGetUserNameindex(t *testing.T) {
	var r string
	testMap.index = make(map[string]map[string]bool) // reset index
	testMap.AddIndex([]string{"t", "x", "UserName", "Tenant"}, &r)
	up := UserProfile{
		Tenant:   "test",
		UserName: "userX",
		Profile: map[string]string{
			"t": "v",
		},
	}
	results := UserProfiles{}
	testMap.GetUsers(up, &results)
	if len(results) != 1 {
		t.Error("error getting users: ", results)
	}
}

func TestUsersGetNotFoundProfileindex(t *testing.T) {
	var r string
	testMap.index = make(map[string]map[string]bool) // reset index
	testMap.AddIndex([]string{"t", "x", "UserName", "Tenant"}, &r)
	up := UserProfile{
		Tenant:   "test",
		UserName: "user",
		Profile: map[string]string{
			"o": "p",
		},
	}
	results := UserProfiles{}
	testMap.GetUsers(up, &results)
	if len(results) != 3 {
		t.Error("error getting users: ", results)
	}
}

func TestUsersGetMissingTenantindex(t *testing.T) {
	var r string
	testMap.index = make(map[string]map[string]bool) // reset index
	testMap.AddIndex([]string{"t", "x", "UserName", "Tenant"}, &r)
	up := UserProfile{
		UserName: "user",
		Profile: map[string]string{
			"t": "v",
		},
	}
	results := UserProfiles{}
	testMap.GetUsers(up, &results)
	if len(results) != 3 {
		t.Error("error getting users: ", results)
	}
}

func TestUsersGetMissingUserNameindex(t *testing.T) {
	var r string
	testMap.index = make(map[string]map[string]bool) // reset index
	testMap.AddIndex([]string{"t", "x", "UserName", "Tenant"}, &r)
	up := UserProfile{
		Tenant: "test",
		Profile: map[string]string{
			"t": "v",
		},
	}
	results := UserProfiles{}
	testMap.GetUsers(up, &results)
	if len(results) != 3 {
		t.Error("error getting users: ", results)
	}
}

func TestUsersGetMissingIdindex(t *testing.T) {
	var r string
	testMap.index = make(map[string]map[string]bool) // reset index
	testMap.AddIndex([]string{"t", "x", "UserName", "Tenant"}, &r)
	up := UserProfile{
		Profile: map[string]string{
			"t": "v",
		},
	}
	results := UserProfiles{}
	testMap.GetUsers(up, &results)
	if len(results) != 4 {
		t.Error("error getting users: ", results)
	}
}

func TestUsersGetMissingIdTwoINdex(t *testing.T) {
	var r string
	testMap.index = make(map[string]map[string]bool) // reset index
	testMap.AddIndex([]string{"t", "x", "UserName", "Tenant"}, &r)
	up := UserProfile{
		Profile: map[string]string{
			"t": "v",
			"x": "y",
		},
	}
	results := UserProfiles{}
	testMap.GetUsers(up, &results)
	if len(results) != 4 {
		t.Error("error getting users: ", results)
	}
}

func TestUsersAddUpdateRemoveIndexes(t *testing.T) {
	tm := newUserMap(accountingStorage, nil)
	var r string
	tm.AddIndex([]string{"t"}, &r)
	if len(tm.index) != 0 {
		t.Error("error adding indexes: ", tm.index)
	}
	tm.SetUser(UserProfile{
		Tenant:   "test",
		UserName: "user",
		Profile: map[string]string{
			"t": "v",
		},
	}, &r)
	if len(tm.index) != 1 || !tm.index["t:v"]["test:user"] {
		t.Error("error adding indexes: ", tm.index)
	}
	tm.SetUser(UserProfile{
		Tenant:   "test",
		UserName: "best",
		Profile: map[string]string{
			"t": "v",
		},
	}, &r)
	if len(tm.index) != 1 ||
		!tm.index["t:v"]["test:user"] ||
		!tm.index["t:v"]["test:best"] {
		t.Error("error adding indexes: ", tm.index)
	}
	tm.UpdateUser(UserProfile{
		Tenant:   "test",
		UserName: "best",
		Profile: map[string]string{
			"t": "v1",
		},
	}, &r)
	if len(tm.index) != 2 ||
		!tm.index["t:v"]["test:user"] ||
		!tm.index["t:v1"]["test:best"] {
		t.Error("error adding indexes: ", tm.index)
	}
	tm.UpdateUser(UserProfile{
		Tenant:   "test",
		UserName: "best",
		Profile: map[string]string{
			"t": "v",
		},
	}, &r)
	if len(tm.index) != 1 ||
		!tm.index["t:v"]["test:user"] ||
		!tm.index["t:v"]["test:best"] {
		t.Error("error adding indexes: ", tm.index)
	}
	tm.RemoveUser(UserProfile{
		Tenant:   "test",
		UserName: "best",
		Profile: map[string]string{
			"t": "v",
		},
	}, &r)
	if len(tm.index) != 1 ||
		!tm.index["t:v"]["test:user"] ||
		tm.index["t:v"]["test:best"] {
		t.Error("error adding indexes: ", tm.index)
	}
	tm.RemoveUser(UserProfile{
		Tenant:   "test",
		UserName: "user",
		Profile: map[string]string{
			"t": "v",
		},
	}, &r)
	if len(tm.index) != 0 {
		t.Error("error adding indexes: ", tm.index)
	}
}

func TestUsersUsageRecordGetLoadUserProfile(t *testing.T) {
	userService = &UserMap{
		table: map[string]map[string]string{
			"test:user":   map[string]string{"TOR": "01", "ReqType": "1", "Direction": "*out", "Category": "c1", "Account": "dan", "Subject": "0723", "Destination": "+401", "SetupTime": "s1", "AnswerTime": "t1", "Usage": "10"},
			":user":       map[string]string{"TOR": "02", "ReqType": "2", "Direction": "*out", "Category": "c2", "Account": "ivo", "Subject": "0724", "Destination": "+402", "SetupTime": "s2", "AnswerTime": "t2", "Usage": "11"},
			"test:":       map[string]string{"TOR": "03", "ReqType": "3", "Direction": "*out", "Category": "c3", "Account": "elloy", "Subject": "0725", "Destination": "+403", "SetupTime": "s3", "AnswerTime": "t3", "Usage": "12"},
			"test1:user1": map[string]string{"TOR": "04", "ReqType": "4", "Direction": "*out", "Category": "call", "Account": "rif", "Subject": "0726", "Destination": "+404", "SetupTime": "s4", "AnswerTime": "t4", "Usage": "13"},
		},
		index: make(map[string]map[string]bool),
	}

	ur := &UsageRecord{
		TOR:         utils.USERS,
		ReqType:     utils.USERS,
		Direction:   "*out",
		Tenant:      "",
		Category:    "call",
		Account:     utils.USERS,
		Subject:     utils.USERS,
		Destination: utils.USERS,
		SetupTime:   utils.USERS,
		AnswerTime:  utils.USERS,
		Usage:       "13",
	}

	out, err := LoadUserProfile(ur, "")
	if err != nil {
		t.Error("Error loading user profile: ", err)
	}
	expected := &UsageRecord{
		TOR:         "04",
		ReqType:     "4",
		Direction:   "*out",
		Tenant:      "",
		Category:    "call",
		Account:     "rif",
		Subject:     "0726",
		Destination: "+404",
		SetupTime:   "s4",
		AnswerTime:  "t4",
		Usage:       "13",
	}
	ur = out.(*UsageRecord)
	if !reflect.DeepEqual(ur, expected) {
		t.Errorf("Expected: %+v got: %+v", expected, ur)
	}
}

func TestUsersExternalCdrGetLoadUserProfileExtraFields(t *testing.T) {
	userService = &UserMap{
		table: map[string]map[string]string{
			"test:user":   map[string]string{"TOR": "01", "ReqType": "1", "Direction": "*out", "Category": "c1", "Account": "dan", "Subject": "0723", "Destination": "+401", "SetupTime": "s1", "AnswerTime": "t1", "Usage": "10"},
			":user":       map[string]string{"TOR": "02", "ReqType": "2", "Direction": "*out", "Category": "c2", "Account": "ivo", "Subject": "0724", "Destination": "+402", "SetupTime": "s2", "AnswerTime": "t2", "Usage": "11"},
			"test:":       map[string]string{"TOR": "03", "ReqType": "3", "Direction": "*out", "Category": "c3", "Account": "elloy", "Subject": "0725", "Destination": "+403", "SetupTime": "s3", "AnswerTime": "t3", "Usage": "12"},
			"test1:user1": map[string]string{"TOR": "04", "ReqType": "4", "Direction": "*out", "Category": "call", "Account": "rif", "Subject": "0726", "Destination": "+404", "SetupTime": "s4", "AnswerTime": "t4", "Usage": "13", "Test": "1"},
		},
		index: make(map[string]map[string]bool),
	}

	ur := &ExternalCdr{
		TOR:         utils.USERS,
		ReqType:     utils.USERS,
		Direction:   "*out",
		Tenant:      "",
		Category:    "call",
		Account:     utils.USERS,
		Subject:     utils.USERS,
		Destination: utils.USERS,
		SetupTime:   utils.USERS,
		AnswerTime:  utils.USERS,
		Usage:       "13",
		ExtraFields: map[string]string{
			"Test": "1",
		},
	}

	out, err := LoadUserProfile(ur, "ExtraFields")
	if err != nil {
		t.Error("Error loading user profile: ", err)
	}
	expected := &ExternalCdr{
		TOR:         "04",
		ReqType:     "4",
		Direction:   "*out",
		Tenant:      "",
		Category:    "call",
		Account:     "rif",
		Subject:     "0726",
		Destination: "+404",
		SetupTime:   "s4",
		AnswerTime:  "t4",
		Usage:       "13",
		ExtraFields: map[string]string{
			"Test": "1",
		},
	}
	ur = out.(*ExternalCdr)
	if !reflect.DeepEqual(ur, expected) {
		t.Errorf("Expected: %+v got: %+v", expected, ur)
	}
}

func TestUsersExternalCdrGetLoadUserProfileExtraFieldsNotFound(t *testing.T) {
	userService = &UserMap{
		table: map[string]map[string]string{
			"test:user":   map[string]string{"TOR": "01", "ReqType": "1", "Direction": "*out", "Category": "c1", "Account": "dan", "Subject": "0723", "Destination": "+401", "SetupTime": "s1", "AnswerTime": "t1", "Usage": "10"},
			":user":       map[string]string{"TOR": "02", "ReqType": "2", "Direction": "*out", "Category": "c2", "Account": "ivo", "Subject": "0724", "Destination": "+402", "SetupTime": "s2", "AnswerTime": "t2", "Usage": "11"},
			"test:":       map[string]string{"TOR": "03", "ReqType": "3", "Direction": "*out", "Category": "c3", "Account": "elloy", "Subject": "0725", "Destination": "+403", "SetupTime": "s3", "AnswerTime": "t3", "Usage": "12"},
			"test1:user1": map[string]string{"TOR": "04", "ReqType": "4", "Direction": "*out", "Category": "call", "Account": "rif", "Subject": "0726", "Destination": "+404", "SetupTime": "s4", "AnswerTime": "t4", "Usage": "13", "Test": "2"},
		},
		index: make(map[string]map[string]bool),
	}

	ur := &ExternalCdr{
		TOR:         utils.USERS,
		ReqType:     utils.USERS,
		Direction:   "*out",
		Tenant:      "",
		Category:    "call",
		Account:     utils.USERS,
		Subject:     utils.USERS,
		Destination: utils.USERS,
		SetupTime:   utils.USERS,
		AnswerTime:  utils.USERS,
		Usage:       "13",
		ExtraFields: map[string]string{
			"Test": "1",
		},
	}

	_, err := LoadUserProfile(ur, "ExtraFields")
	if err != utils.ErrNotFound {
		t.Error("Error detecting err in loading user profile: ", err)
	}
}

func TestUsersExternalCdrGetLoadUserProfileExtraFieldsSet(t *testing.T) {
	userService = &UserMap{
		table: map[string]map[string]string{
			"test:user":   map[string]string{"TOR": "01", "ReqType": "1", "Direction": "*out", "Category": "c1", "Account": "dan", "Subject": "0723", "Destination": "+401", "SetupTime": "s1", "AnswerTime": "t1", "Usage": "10"},
			":user":       map[string]string{"TOR": "02", "ReqType": "2", "Direction": "*out", "Category": "c2", "Account": "ivo", "Subject": "0724", "Destination": "+402", "SetupTime": "s2", "AnswerTime": "t2", "Usage": "11"},
			"test:":       map[string]string{"TOR": "03", "ReqType": "3", "Direction": "*out", "Category": "c3", "Account": "elloy", "Subject": "0725", "Destination": "+403", "SetupTime": "s3", "AnswerTime": "t3", "Usage": "12"},
			"test1:user1": map[string]string{"TOR": "04", "ReqType": "4", "Direction": "*out", "Category": "call", "Account": "rif", "Subject": "0726", "Destination": "+404", "SetupTime": "s4", "AnswerTime": "t4", "Usage": "13", "Test": "1", "Best": "BestValue"},
		},
		index: make(map[string]map[string]bool),
	}

	ur := &ExternalCdr{
		TOR:         utils.USERS,
		ReqType:     utils.USERS,
		Direction:   "*out",
		Tenant:      "",
		Category:    "call",
		Account:     utils.USERS,
		Subject:     utils.USERS,
		Destination: utils.USERS,
		SetupTime:   utils.USERS,
		AnswerTime:  utils.USERS,
		Usage:       "13",
		ExtraFields: map[string]string{
			"Test": "1",
			"Best": utils.USERS,
		},
	}

	out, err := LoadUserProfile(ur, "ExtraFields")
	if err != nil {
		t.Error("Error loading user profile: ", err)
	}
	expected := &ExternalCdr{
		TOR:         "04",
		ReqType:     "4",
		Direction:   "*out",
		Tenant:      "",
		Category:    "call",
		Account:     "rif",
		Subject:     "0726",
		Destination: "+404",
		SetupTime:   "s4",
		AnswerTime:  "t4",
		Usage:       "13",
		ExtraFields: map[string]string{
			"Test": "1",
			"Best": "BestValue",
		},
	}
	ur = out.(*ExternalCdr)
	if !reflect.DeepEqual(ur, expected) {
		t.Errorf("Expected: %+v got: %+v", expected, ur)
	}
}

func TestUsersCallDescLoadUserProfile(t *testing.T) {
	userService = &UserMap{
		table: map[string]map[string]string{
			"cgrates.org:dan":      map[string]string{"ReqType": "*prepaid", "Category": "call1", "Account": "dan", "Subject": "dan", "Cli": "+4986517174963"},
			"cgrates.org:danvoice": map[string]string{"TOR": "*voice", "ReqType": "*prepaid", "Category": "call1", "Account": "dan", "Subject": "0723"},
			"cgrates:rif":          map[string]string{"ReqType": "*postpaid", "Direction": "*out", "Category": "call", "Account": "rif", "Subject": "0726"},
		},
		index: make(map[string]map[string]bool),
	}
	startTime := time.Now()
	cd := &CallDescriptor{
		TOR:         "*sms",
		Tenant:      utils.USERS,
		Category:    utils.USERS,
		Subject:     utils.USERS,
		Account:     utils.USERS,
		Destination: "+4986517174963",
		TimeStart:   startTime,
		TimeEnd:     startTime.Add(time.Duration(1) * time.Minute),
		ExtraFields: map[string]string{"Cli": "+4986517174963"},
	}
	expected := &CallDescriptor{
		TOR:         "*sms",
		Tenant:      "cgrates.org",
		Category:    "call1",
		Account:     "dan",
		Subject:     "dan",
		Destination: "+4986517174963",
		TimeStart:   startTime,
		TimeEnd:     startTime.Add(time.Duration(1) * time.Minute),
		ExtraFields: map[string]string{"Cli": "+4986517174963"},
	}
	out, err := LoadUserProfile(cd, "ExtraFields")
	if err != nil {
		t.Error("Error loading user profile: ", err)
	}
	cdRcv := out.(*CallDescriptor)
	if !reflect.DeepEqual(expected, cdRcv) {
		t.Errorf("Expected: %+v got: %+v", expected, cdRcv)
	}
}

func TestUsersStoredCdrLoadUserProfile(t *testing.T) {
	userService = &UserMap{
		table: map[string]map[string]string{
			"cgrates.org:dan":      map[string]string{"ReqType": "*prepaid", "Category": "call1", "Account": "dan", "Subject": "dan", "Cli": "+4986517174963"},
			"cgrates.org:danvoice": map[string]string{"TOR": "*voice", "ReqType": "*prepaid", "Category": "call1", "Account": "dan", "Subject": "0723"},
			"cgrates:rif":          map[string]string{"ReqType": "*postpaid", "Direction": "*out", "Category": "call", "Account": "rif", "Subject": "0726"},
		},
		index: make(map[string]map[string]bool),
	}
	startTime := time.Now()
	cdr := &StoredCdr{
		TOR:         "*sms",
		ReqType:     utils.USERS,
		Tenant:      utils.USERS,
		Category:    utils.USERS,
		Account:     utils.USERS,
		Subject:     utils.USERS,
		Destination: "+4986517174963",
		SetupTime:   startTime,
		AnswerTime:  startTime,
		Usage:       time.Duration(1) * time.Minute,
		ExtraFields: map[string]string{"Cli": "+4986517174963"},
	}
	expected := &StoredCdr{
		TOR:         "*sms",
		ReqType:     "*prepaid",
		Tenant:      "cgrates.org",
		Category:    "call1",
		Account:     "dan",
		Subject:     "dan",
		Destination: "+4986517174963",
		SetupTime:   startTime,
		AnswerTime:  startTime,
		Usage:       time.Duration(1) * time.Minute,
		ExtraFields: map[string]string{"Cli": "+4986517174963"},
	}
	out, err := LoadUserProfile(cdr, "ExtraFields")
	if err != nil {
		t.Error("Error loading user profile: ", err)
	}
	cdRcv := out.(*StoredCdr)
	if !reflect.DeepEqual(expected, cdRcv) {
		t.Errorf("Expected: %+v got: %+v", expected, cdRcv)
	}
}
