package engine

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/cgrates/cgrates/utils"
	"github.com/cgrates/rpcclient"
)

type UserProfile struct {
	Tenant   string
	UserName string
	Profile  map[string]string
	ponder   int
}

type UserProfiles []*UserProfile

func (ups UserProfiles) Len() int {
	return len(ups)
}

func (ups UserProfiles) Swap(i, j int) {
	ups[i], ups[j] = ups[j], ups[i]
}

func (ups UserProfiles) Less(j, i int) bool { // get higher ponder in front
	return ups[i].ponder < ups[j].ponder
}

func (ups UserProfiles) Sort() {
	sort.Sort(ups)
}

func (ud *UserProfile) GetId() string {
	return utils.ConcatenatedKey(ud.Tenant, ud.UserName)
}

func (ud *UserProfile) SetId(id string) error {
	vals := strings.Split(id, utils.CONCATENATED_KEY_SEP)
	if len(vals) != 2 {
		return utils.ErrInvalidKey
	}
	ud.Tenant = vals[0]
	ud.UserName = vals[1]
	return nil
}

type UserService interface {
	SetUser(UserProfile, *string) error
	RemoveUser(UserProfile, *string) error
	UpdateUser(UserProfile, *string) error
	GetUsers(UserProfile, *UserProfiles) error
	AddIndex([]string, *string) error
	GetIndexes(string, *map[string][]string) error
	ReloadUsers(string, *string) error
}

type UserMap struct {
	table        map[string]map[string]string
	index        map[string]map[string]bool
	indexKeys    []string
	accountingDb AccountingStorage
	mu           sync.RWMutex
}

func NewUserMap(accountingDb AccountingStorage, indexes []string) (*UserMap, error) {
	um := newUserMap(accountingDb, indexes)
	var reply string
	if err := um.ReloadUsers("", &reply); err != nil {
		return nil, err
	}
	return um, nil
}

func newUserMap(accountingDb AccountingStorage, indexes []string) *UserMap {
	return &UserMap{
		table:        make(map[string]map[string]string),
		index:        make(map[string]map[string]bool),
		indexKeys:    indexes,
		accountingDb: accountingDb,
	}
}

func (um *UserMap) ReloadUsers(in string, reply *string) error {
	um.mu.Lock()

	// backup old data
	oldTable := um.table
	oldIndex := um.index
	um.table = make(map[string]map[string]string)
	um.index = make(map[string]map[string]bool)

	// load from rating db
	if ups, err := um.accountingDb.GetUsers(); err == nil {
		for _, up := range ups {
			um.table[up.GetId()] = up.Profile
		}
	} else {
		// restore old data before return
		um.table = oldTable
		um.index = oldIndex

		*reply = err.Error()
		return err
	}
	um.mu.Unlock()

	if len(um.indexKeys) != 0 {
		var s string
		if err := um.AddIndex(um.indexKeys, &s); err != nil {
			Logger.Err(fmt.Sprintf("Error adding %v indexes to user profile service: %v", um.indexKeys, err))
			um.table = oldTable
			um.index = oldIndex

			*reply = err.Error()
			return err
		}
	}
	*reply = utils.OK
	return nil
}

func (um *UserMap) SetUser(up UserProfile, reply *string) error {
	um.mu.Lock()
	defer um.mu.Unlock()
	if err := um.accountingDb.SetUser(&up); err != nil {
		*reply = err.Error()
		return err
	}
	um.table[up.GetId()] = up.Profile
	um.addIndex(&up, um.indexKeys)
	*reply = utils.OK
	return nil
}

func (um *UserMap) RemoveUser(up UserProfile, reply *string) error {
	um.mu.Lock()
	defer um.mu.Unlock()
	if err := um.accountingDb.RemoveUser(up.GetId()); err != nil {
		*reply = err.Error()
		return err
	}
	delete(um.table, up.GetId())
	um.deleteIndex(&up)
	*reply = utils.OK
	return nil
}

func (um *UserMap) UpdateUser(up UserProfile, reply *string) error {
	um.mu.Lock()
	defer um.mu.Unlock()
	m, found := um.table[up.GetId()]
	if !found {
		*reply = utils.ErrNotFound.Error()
		return utils.ErrNotFound
	}
	if m == nil {
		m = make(map[string]string)
	}
	oldM := make(map[string]string, len(m))
	for k, v := range m {
		oldM[k] = v
	}
	oldUp := &UserProfile{
		Tenant:   up.Tenant,
		UserName: up.UserName,
		Profile:  oldM,
	}
	for key, value := range up.Profile {
		m[key] = value
	}
	finalUp := &UserProfile{
		Tenant:   up.Tenant,
		UserName: up.UserName,
		Profile:  m,
	}
	if err := um.accountingDb.SetUser(finalUp); err != nil {
		*reply = err.Error()
		return err
	}
	um.table[up.GetId()] = m
	um.deleteIndex(oldUp)
	um.addIndex(finalUp, um.indexKeys)
	*reply = utils.OK
	return nil
}

func (um *UserMap) GetUsers(up UserProfile, results *UserProfiles) error {
	um.mu.RLock()
	defer um.mu.RUnlock()
	table := um.table // no index

	indexUnionKeys := make(map[string]bool)
	// search index
	if up.Tenant != "" {
		if keys, found := um.index[utils.ConcatenatedKey("Tenant", up.Tenant)]; found {
			for key := range keys {
				indexUnionKeys[key] = true
			}
		}
	}
	if up.UserName != "" {
		if keys, found := um.index[utils.ConcatenatedKey("UserName", up.UserName)]; found {
			for key := range keys {
				indexUnionKeys[key] = true
			}
		}
	}
	for k, v := range up.Profile {
		if keys, found := um.index[utils.ConcatenatedKey(k, v)]; found {
			for key := range keys {
				indexUnionKeys[key] = true
			}
		}
	}
	if len(indexUnionKeys) != 0 {
		table = make(map[string]map[string]string)
		for key := range indexUnionKeys {
			table[key] = um.table[key]
		}
	}

	candidates := make(UserProfiles, 0) // It should not return nil in case of no users but []
	for key, values := range table {
		ponder := 0
		tableUP := &UserProfile{
			Profile: values,
		}
		tableUP.SetId(key)
		if up.Tenant != "" && tableUP.Tenant != "" && up.Tenant != tableUP.Tenant {
			continue
		}
		if tableUP.Tenant != "" {
			ponder += 1
		}
		if up.UserName != "" && tableUP.UserName != "" && up.UserName != tableUP.UserName {
			continue
		}
		if tableUP.UserName != "" {
			ponder += 1
		}
		valid := true
		for k, v := range up.Profile {
			if tableUP.Profile[k] != "" && tableUP.Profile[k] != v {
				valid = false
				break
			}
			if tableUP.Profile[k] != "" {
				ponder += 1
			}
		}
		if !valid {
			continue
		}
		// all filters passed, add to candidates
		nup := &UserProfile{Profile: make(map[string]string)}
		nup.SetId(key)
		nup.ponder = ponder
		for k, v := range tableUP.Profile {
			nup.Profile[k] = v
		}
		candidates = append(candidates, nup)
	}
	candidates.Sort()
	*results = candidates
	return nil
}

func (um *UserMap) AddIndex(indexes []string, reply *string) error {
	um.mu.Lock()
	defer um.mu.Unlock()
	um.indexKeys = append(um.indexKeys, indexes...)
	for key, values := range um.table {
		up := &UserProfile{Profile: values}
		up.SetId(key)
		um.addIndex(up, indexes)
	}
	*reply = utils.OK
	return nil
}

func (um *UserMap) addIndex(up *UserProfile, indexes []string) {
	key := up.GetId()
	for _, index := range indexes {
		if index == "Tenant" {
			if up.Tenant != "" {
				indexKey := utils.ConcatenatedKey(index, up.Tenant)
				if um.index[indexKey] == nil {
					um.index[indexKey] = make(map[string]bool)
				}
				um.index[indexKey][key] = true
			}
			continue
		}
		if index == "UserName" {
			if up.UserName != "" {
				indexKey := utils.ConcatenatedKey(index, up.UserName)
				if um.index[indexKey] == nil {
					um.index[indexKey] = make(map[string]bool)
				}
				um.index[indexKey][key] = true
			}
			continue
		}

		for k, v := range up.Profile {
			if k == index && v != "" {
				indexKey := utils.ConcatenatedKey(k, v)
				if um.index[indexKey] == nil {
					um.index[indexKey] = make(map[string]bool)
				}
				um.index[indexKey][key] = true
			}
		}
	}
}

func (um *UserMap) deleteIndex(up *UserProfile) {
	key := up.GetId()
	for _, index := range um.indexKeys {
		if index == "Tenant" {
			if up.Tenant != "" {
				indexKey := utils.ConcatenatedKey(index, up.Tenant)
				delete(um.index[indexKey], key)
				if len(um.index[indexKey]) == 0 {
					delete(um.index, indexKey)
				}
			}
			continue
		}
		if index == "UserName" {
			if up.UserName != "" {
				indexKey := utils.ConcatenatedKey(index, up.UserName)
				delete(um.index[indexKey], key)
				if len(um.index[indexKey]) == 0 {
					delete(um.index, indexKey)
				}
			}
			continue
		}
		for k, v := range up.Profile {
			if k == index && v != "" {
				indexKey := utils.ConcatenatedKey(k, v)
				delete(um.index[indexKey], key)
				if len(um.index[indexKey]) == 0 {
					delete(um.index, indexKey)
				}
			}
		}
	}
}

func (um *UserMap) GetIndexes(in string, reply *map[string][]string) error {
	um.mu.RLock()
	defer um.mu.RUnlock()
	indexes := make(map[string][]string)
	for key, values := range um.index {
		var vs []string
		for val := range values {
			vs = append(vs, val)
		}
		indexes[key] = vs
	}
	*reply = indexes
	return nil
}

type UserProxy struct{}

type ProxyUserService struct {
	Client *rpcclient.RpcClient
}

func NewProxyUserService(addr string, attempts, reconnects int) (*ProxyUserService, error) {
	client, err := rpcclient.NewRpcClient("tcp", addr, attempts, reconnects, utils.GOB)
	if err != nil {
		return nil, err
	}
	return &ProxyUserService{Client: client}, nil
}

func (ps *ProxyUserService) SetUser(ud UserProfile, reply *string) error {
	return ps.Client.Call("UsersV1.SetUser", ud, reply)
}

func (ps *ProxyUserService) RemoveUser(ud UserProfile, reply *string) error {
	return ps.Client.Call("UsersV1.RemoveUser", ud, reply)
}

func (ps *ProxyUserService) UpdateUser(ud UserProfile, reply *string) error {
	return ps.Client.Call("UsersV1.UpdateUser", ud, reply)
}

func (ps *ProxyUserService) GetUsers(ud UserProfile, users *UserProfiles) error {
	return ps.Client.Call("UsersV1.GetUsers", ud, users)
}

func (ps *ProxyUserService) AddIndex(indexes []string, reply *string) error {
	return ps.Client.Call("UsersV1.AddIndex", indexes, reply)
}

func (ps *ProxyUserService) GetIndexes(in string, reply *map[string][]string) error {
	return ps.Client.Call("UsersV1.AddIndex", in, reply)
}

func (ps *ProxyUserService) ReloadUsers(in string, reply *string) error {
	return ps.Client.Call("UsersV1.ReloadUsers", in, reply)
}

// extraFields - Field name in the interface containing extraFields information
func LoadUserProfile(in interface{}, extraFields string) (interface{}, error) {
	if userService == nil { // no user service => no fun
		return in, nil
	}
	m := utils.ToMapStringString(in)
	var needsUsers bool
	for _, val := range m {
		if val == utils.USERS {
			needsUsers = true
			break
		}
	}
	if !needsUsers { // Do not process further if user profile is not needed
		return in, nil
	}
	up := &UserProfile{
		Profile: make(map[string]string),
	}
	tenant := m["Tenant"]
	if tenant != "" && tenant != utils.USERS {
		up.Tenant = tenant
	}
	delete(m, "Tenant")

	// clean empty and *user fields
	for key, val := range m {
		if val != "" && val != utils.USERS {
			up.Profile[key] = val
		}
	}
	// add extra fields
	if extraFields != "" {
		extra := utils.GetMapExtraFields(in, extraFields)
		for key, val := range extra {
			if val != "" && val != utils.USERS {
				up.Profile[key] = val
			}
		}
	}
	ups := UserProfiles{}
	if err := userService.GetUsers(*up, &ups); err != nil {
		return nil, err
	}
	if len(ups) > 0 {
		up = ups[0]
		m := up.Profile
		m["Tenant"] = up.Tenant
		utils.FromMapStringString(m, in)
		utils.SetMapExtraFields(in, m, extraFields)
		return in, nil
	}
	return nil, utils.ErrNotFound
}
