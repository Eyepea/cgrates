/*
Real-time Charging System for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

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

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cgrates/cgrates/utils"
)

const (
	DISABLED = "disabled"
	JSON     = "json"
	GOB      = "gob"
	POSTGRES = "postgres"
	MONGO    = "mongo"
	REDIS    = "redis"
	SAME     = "same"
	FS       = "freeswitch"
)

var (
	cgrCfg            *CGRConfig     // will be shared
	dfltFsConnConfig  *FsConnConfig  // Default FreeSWITCH Connection configuration, built out of json default configuration
	dfltKamConnConfig *KamConnConfig // Default Kamailio Connection configuration
)

// Used to retrieve system configuration from other packages
func CgrConfig() *CGRConfig {
	return cgrCfg
}

// Used to set system configuration from other places
func SetCgrConfig(cfg *CGRConfig) {
	cgrCfg = cfg
}

func NewDefaultCGRConfig() (*CGRConfig, error) {
	cfg := new(CGRConfig)
	cfg.DataFolderPath = "/usr/share/cgrates/"
	cfg.SmFsConfig = new(SmFsConfig)
	cfg.SmKamConfig = new(SmKamConfig)
	cfg.SmOsipsConfig = new(SmOsipsConfig)
	cfg.ConfigReloads = make(map[string]chan struct{})
	cfg.ConfigReloads[utils.CDRC] = make(chan struct{})
	cgrJsonCfg, err := NewCgrJsonCfgFromReader(strings.NewReader(CGRATES_CFG_JSON))
	if err != nil {
		return nil, err
	}
	cfg.MaxCallDuration = time.Duration(3) * time.Hour // Hardcoded for now
	if err := cfg.loadFromJsonCfg(cgrJsonCfg); err != nil {
		return nil, err
	}
	cfg.dfltCdreProfile = cfg.CdreProfiles[utils.META_DEFAULT].Clone() // So default will stay unique, will have nil pointer in case of no defaults loaded which is an extra check
	cfg.dfltCdrcProfile = cfg.CdrcProfiles["/var/log/cgrates/cdrc/in"][utils.META_DEFAULT].Clone()
	dfltFsConnConfig = cfg.SmFsConfig.Connections[0] // We leave it crashing here on purpose if no Connection defaults defined
	dfltKamConnConfig = cfg.SmKamConfig.Connections[0]
	if err := cfg.checkConfigSanity(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func NewCGRConfigFromJsonString(cfgJsonStr string) (*CGRConfig, error) {
	cfg := new(CGRConfig)
	if jsnCfg, err := NewCgrJsonCfgFromReader(strings.NewReader(cfgJsonStr)); err != nil {
		return nil, err
	} else if err := cfg.loadFromJsonCfg(jsnCfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func NewCGRConfigFromJsonStringWithDefaults(cfgJsonStr string) (*CGRConfig, error) {
	cfg, _ := NewDefaultCGRConfig()
	if jsnCfg, err := NewCgrJsonCfgFromReader(strings.NewReader(cfgJsonStr)); err != nil {
		return nil, err
	} else if err := cfg.loadFromJsonCfg(jsnCfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Reads all .json files out of a folder/subfolders and loads them up in lexical order
func NewCGRConfigFromFolder(cfgDir string) (*CGRConfig, error) {
	cfg, err := NewDefaultCGRConfig()
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(cfgDir)
	if err != nil {
		if strings.HasSuffix(err.Error(), "no such file or directory") {
			return cfg, nil
		}
		return nil, err
	} else if !fi.IsDir() && cfgDir != utils.CONFIG_DIR { // If config dir defined, needs to exist, not checking for default
		return nil, fmt.Errorf("Path: %s not a directory.", cfgDir)
	}
	if fi.IsDir() {
		jsonFilesFound := false
		err = filepath.Walk(cfgDir, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				return nil
			}
			cfgFiles, err := filepath.Glob(filepath.Join(path, "*.json"))
			if err != nil {
				return err
			}
			if cfgFiles == nil { // No need of processing further since there are no config files in the folder
				return nil
			}
			if !jsonFilesFound {
				jsonFilesFound = true
			}
			for _, jsonFilePath := range cfgFiles {
				if cgrJsonCfg, err := NewCgrJsonCfgFromFile(jsonFilePath); err != nil {
					return err
				} else if err := cfg.loadFromJsonCfg(cgrJsonCfg); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		} else if !jsonFilesFound {
			return nil, fmt.Errorf("No config file found on path %s", cfgDir)
		}
	}
	if err := cfg.checkConfigSanity(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Holds system configuration, defaults are overwritten with values from config file if found
type CGRConfig struct {
	TpDbType             string
	TpDbHost             string // The host to connect to. Values that start with / are for UNIX domain sockets.
	TpDbPort             string // The port to bind to.
	TpDbName             string // The name of the database to connect to.
	TpDbUser             string // The user to sign in as.
	TpDbPass             string // The user's password.
	DataDbType           string
	DataDbHost           string        // The host to connect to. Values that start with / are for UNIX domain sockets.
	DataDbPort           string        // The port to bind to.
	DataDbName           string        // The name of the database to connect to.
	DataDbUser           string        // The user to sign in as.
	DataDbPass           string        // The user's password.
	StorDBType           string        // Should reflect the database type used to store logs
	StorDBHost           string        // The host to connect to. Values that start with / are for UNIX domain sockets.
	StorDBPort           string        // Th e port to bind to.
	StorDBName           string        // The name of the database to connect to.
	StorDBUser           string        // The user to sign in as.
	StorDBPass           string        // The user's password.
	StorDBMaxOpenConns   int           // Maximum database connections opened
	StorDBMaxIdleConns   int           // Maximum idle connections to keep opened
	DBDataEncoding       string        // The encoding used to store object data in strings: <msgpack|json>
	RPCJSONListen        string        // RPC JSON listening address
	RPCGOBListen         string        // RPC GOB listening address
	HTTPListen           string        // HTTP listening address
	DefaultReqType       string        // Use this request type if not defined on top
	DefaultCategory      string        // set default type of record
	DefaultTenant        string        // set default tenant
	DefaultSubject       string        // set default rating subject, useful in case of fallback
	Reconnects           int           // number of recconect attempts in case of connection lost <-1 for infinite | nb>
	ConnectAttempts      int           // number of initial connection attempts before giving up
	RoundingDecimals     int           // Number of decimals to round end prices at
	HttpSkipTlsVerify    bool          // If enabled Http Client will accept any TLS certificate
	TpExportPath         string        // Path towards export folder for offline Tariff Plans
	MaxCallDuration      time.Duration // The maximum call duration (used by responder when querying DerivedCharging) // ToDo: export it in configuration file
	RaterEnabled         bool          // start standalone server (no balancer)
	RaterBalancer        string        // balancer address host:port
	RaterCdrStats        string        // address where to reach the cdrstats service. Empty to disable stats gathering  <""|internal|x.y.z.y:1234>
	RaterHistoryServer   string
	RaterPubSubServer    string
	RaterUserServer      string
	BalancerEnabled      bool
	SchedulerEnabled     bool
	CDRSEnabled          bool                 // Enable CDR Server service
	CDRSExtraFields      []*utils.RSRField    // Extra fields to store in CDRs
	CDRSStoreCdrs        bool                 // store cdrs in storDb
	CDRSRater            string               // address where to reach the Rater for cost calculation: <""|internal|x.y.z.y:1234>
	CDRSStats            string               // address where to reach the cdrstats service. Empty to disable stats gathering  <""|internal|x.y.z.y:1234>
	CDRSReconnects       int                  // number of reconnects to remote services before giving up
	CDRSCdrReplication   []*CdrReplicationCfg // Replicate raw CDRs to a number of servers
	CDRStatsEnabled      bool                 // Enable CDR Stats service
	CDRStatsSaveInterval time.Duration        // Save interval duration
	//CDRStatConfig        *CdrStatsConfig // Active cdr stats configuration instances, platform level
	CdreProfiles         map[string]*CdreConfig
	CdrcProfiles         map[string]map[string]*CdrcConfig // Number of CDRC instances running imports, format map[dirPath]map[instanceName]{Configs}
	SmFsConfig           *SmFsConfig                       // SM-FreeSWITCH configuration
	SmKamConfig          *SmKamConfig                      // SM-Kamailio Configuration
	SmOsipsConfig        *SmOsipsConfig                    // SM-OpenSIPS Configuration
	HistoryServer        string                            // Address where to reach the master history server: <internal|x.y.z.y:1234>
	HistoryServerEnabled bool                              // Starts History as server: <true|false>.
	HistoryDir           string                            // Location on disk where to store history files.
	HistorySaveInterval  time.Duration                     // The timout duration between pubsub writes
	PubSubServer         string                            // Address where to reach the master pubsub server: <internal|x.y.z.y:1234>
	PubSubServerEnabled  bool                              // Starts PubSub as server: <true|false>.
	UserServerEnabled    bool                              // Starts User as server: <true|false>
	UserServerIndexes    []string                          // List of user profile field indexes
	MailerServer         string                            // The server to use when sending emails out
	MailerAuthUser       string                            // Authenticate to email server using this user
	MailerAuthPass       string                            // Authenticate to email server with this password
	MailerFromAddr       string                            // From address used when sending emails out
	DataFolderPath       string                            // Path towards data folder, for tests internal usage, not loading out of .json options
	ConfigReloads        map[string]chan struct{}          // Signals to specific entities that a config reload should occur
	// Cache defaults loaded from json and needing clones
	dfltCdreProfile *CdreConfig // Default cdreConfig profile
	dfltCdrcProfile *CdrcConfig // Default cdrcConfig profile

}

func (self *CGRConfig) checkConfigSanity() error {
	// Rater checks
	if self.RaterEnabled {
		if self.RaterBalancer == utils.INTERNAL && !self.BalancerEnabled {
			return errors.New("Balancer not enabled but requested by Rater component.")
		}
		if self.RaterCdrStats == utils.INTERNAL && !self.CDRStatsEnabled {
			return errors.New("CDRStats not enabled but requested by Rater component.")
		}
		if self.RaterHistoryServer == utils.INTERNAL && !self.HistoryServerEnabled {
			return errors.New("History server not enabled but requested by Rater component.")
		}
		if self.RaterPubSubServer == utils.INTERNAL && !self.PubSubServerEnabled {
			return errors.New("PubSub server not enabled but requested by Rater component.")
		}
		if self.RaterUserServer == utils.INTERNAL && !self.UserServerEnabled {
			return errors.New("Users service not enabled but requested by Rater component.")
		}
	}
	// CDRServer checks
	if self.CDRSEnabled {
		if self.CDRSRater == utils.INTERNAL && !self.RaterEnabled {
			return errors.New("Rater not enabled but requested by CDRS component.")
		}
		if self.CDRSStats == utils.INTERNAL && !self.CDRStatsEnabled {
			return errors.New("CDRStats not enabled but requested by CDRS component.")
		}
	}
	// CDRC sanity checks
	for _, cdrcCfgs := range self.CdrcProfiles {
		for _, cdrcInst := range cdrcCfgs {
			if !cdrcInst.Enabled {
				continue
			}
			if len(cdrcInst.Cdrs) == 0 {
				return errors.New("CdrC enabled but no CDRS defined!")
			} else if cdrcInst.Cdrs == utils.INTERNAL && !self.CDRSEnabled {
				return errors.New("CDRS not enabled but referenced from CDRC")
			}
			if len(cdrcInst.ContentFields) == 0 {
				return errors.New("CdrC enabled but no fields to be processed defined!")
			}
			if cdrcInst.CdrFormat == utils.CSV {
				for _, cdrFld := range cdrcInst.ContentFields {
					for _, rsrFld := range cdrFld.Value {
						if _, errConv := strconv.Atoi(rsrFld.Id); errConv != nil && !rsrFld.IsStatic() {
							return fmt.Errorf("CDR fields must be indices in case of .csv files, have instead: %s", rsrFld.Id)
						}
					}
				}
			}
		}
	}
	// SM-FreeSWITCH checks
	if self.SmFsConfig.Enabled {
		if self.SmFsConfig.Rater == "" {
			return errors.New("Rater definition is mandatory!")
		}
		if self.SmFsConfig.Cdrs == "" {
			return errors.New("Cdrs definition is mandatory!")
		}
		if self.SmFsConfig.Rater == utils.INTERNAL && !self.RaterEnabled {
			return errors.New("Rater not enabled but requested by SM-FreeSWITCH component.")
		}
		if self.SmFsConfig.Cdrs == utils.INTERNAL && !self.CDRSEnabled {
			return errors.New("CDRS not enabled but referenced by SM-FreeSWITCH component")
		}
	}
	// SM-Kamailio checks
	if self.SmKamConfig.Enabled {
		if self.SmKamConfig.Rater == "" {
			return errors.New("Rater definition is mandatory!")
		}
		if self.SmKamConfig.Cdrs == "" {
			return errors.New("Cdrs definition is mandatory!")
		}
		if self.SmKamConfig.Rater == utils.INTERNAL && !self.RaterEnabled {
			return errors.New("Rater not enabled but requested by SM-Kamailio component.")
		}
		if self.SmKamConfig.Cdrs == utils.INTERNAL && !self.CDRSEnabled {
			return errors.New("CDRS not enabled but referenced by SM-Kamailio component")
		}
	}
	// SM-OpenSIPS checks
	if self.SmOsipsConfig.Enabled {
		if self.SmOsipsConfig.Rater == "" {
			return errors.New("Rater definition is mandatory!")
		}
		if self.SmOsipsConfig.Cdrs == "" {
			return errors.New("Cdrs definition is mandatory!")
		}
		if self.SmOsipsConfig.Rater == utils.INTERNAL && !self.RaterEnabled {
			return errors.New("Rater not enabled but requested by SM-OpenSIPS component.")
		}
		if self.SmOsipsConfig.Cdrs == utils.INTERNAL && !self.CDRSEnabled {
			return errors.New("CDRS not enabled but referenced by SM-OpenSIPS component")
		}
	}
	return nil
}

// Loads from json configuration object, will be used for defaults, config from file and reload, might need lock
func (self *CGRConfig) loadFromJsonCfg(jsnCfg *CgrJsonCfg) error {

	// Load sections out of JSON config, stop on error
	jsnGeneralCfg, err := jsnCfg.GeneralJsonCfg()
	if err != nil {
		return err
	}

	jsnListenCfg, err := jsnCfg.ListenJsonCfg()
	if err != nil {
		return err
	}

	jsnTpDbCfg, err := jsnCfg.DbJsonCfg(TPDB_JSN)
	if err != nil {
		return err
	}

	jsnDataDbCfg, err := jsnCfg.DbJsonCfg(DATADB_JSN)
	if err != nil {
		return err
	}

	jsnStorDbCfg, err := jsnCfg.DbJsonCfg(STORDB_JSN)
	if err != nil {
		return err
	}

	jsnBalancerCfg, err := jsnCfg.BalancerJsonCfg()
	if err != nil {
		return err
	}

	jsnRaterCfg, err := jsnCfg.RaterJsonCfg()
	if err != nil {
		return err
	}

	jsnSchedCfg, err := jsnCfg.SchedulerJsonCfg()
	if err != nil {
		return err
	}

	jsnCdrsCfg, err := jsnCfg.CdrsJsonCfg()
	if err != nil {
		return err
	}

	jsnCdrstatsCfg, err := jsnCfg.CdrStatsJsonCfg()
	if err != nil {
		return err
	}

	jsnCdreCfg, err := jsnCfg.CdreJsonCfgs()
	if err != nil {
		return err
	}

	jsnCdrcCfg, err := jsnCfg.CdrcJsonCfg()
	if err != nil {
		return err
	}

	jsnSmFsCfg, err := jsnCfg.SmFsJsonCfg()
	if err != nil {
		return err
	}

	jsnSmKamCfg, err := jsnCfg.SmKamJsonCfg()
	if err != nil {
		return err
	}

	jsnSmOsipsCfg, err := jsnCfg.SmOsipsJsonCfg()
	if err != nil {
		return err
	}

	jsnHistServCfg, err := jsnCfg.HistServJsonCfg()
	if err != nil {
		return err
	}

	jsnPubSubServCfg, err := jsnCfg.PubSubServJsonCfg()
	if err != nil {
		return err
	}

	jsnUserServCfg, err := jsnCfg.UserServJsonCfg()
	if err != nil {
		return err
	}

	jsnMailerCfg, err := jsnCfg.MailerJsonCfg()
	if err != nil {
		return err
	}

	// All good, start populating config variables
	if jsnTpDbCfg != nil {
		if jsnTpDbCfg.Db_type != nil {
			self.TpDbType = *jsnTpDbCfg.Db_type
		}
		if jsnTpDbCfg.Db_host != nil {
			self.TpDbHost = *jsnTpDbCfg.Db_host
		}
		if jsnTpDbCfg.Db_port != nil {
			self.TpDbPort = strconv.Itoa(*jsnTpDbCfg.Db_port)
		}
		if jsnTpDbCfg.Db_name != nil {
			self.TpDbName = *jsnTpDbCfg.Db_name
		}
		if jsnTpDbCfg.Db_user != nil {
			self.TpDbUser = *jsnTpDbCfg.Db_user
		}
		if jsnTpDbCfg.Db_passwd != nil {
			self.TpDbPass = *jsnTpDbCfg.Db_passwd
		}
	}

	if jsnDataDbCfg != nil {
		if jsnDataDbCfg.Db_type != nil {
			self.DataDbType = *jsnDataDbCfg.Db_type
		}
		if jsnDataDbCfg.Db_host != nil {
			self.DataDbHost = *jsnDataDbCfg.Db_host
		}
		if jsnDataDbCfg.Db_port != nil {
			self.DataDbPort = strconv.Itoa(*jsnDataDbCfg.Db_port)
		}
		if jsnDataDbCfg.Db_name != nil {
			self.DataDbName = *jsnDataDbCfg.Db_name
		}
		if jsnDataDbCfg.Db_user != nil {
			self.DataDbUser = *jsnDataDbCfg.Db_user
		}
		if jsnDataDbCfg.Db_passwd != nil {
			self.DataDbPass = *jsnDataDbCfg.Db_passwd
		}
	}

	if jsnStorDbCfg != nil {
		if jsnStorDbCfg.Db_type != nil {
			self.StorDBType = *jsnStorDbCfg.Db_type
		}
		if jsnStorDbCfg.Db_host != nil {
			self.StorDBHost = *jsnStorDbCfg.Db_host
		}
		if jsnStorDbCfg.Db_port != nil {
			self.StorDBPort = strconv.Itoa(*jsnStorDbCfg.Db_port)
		}
		if jsnStorDbCfg.Db_name != nil {
			self.StorDBName = *jsnStorDbCfg.Db_name
		}
		if jsnStorDbCfg.Db_user != nil {
			self.StorDBUser = *jsnStorDbCfg.Db_user
		}
		if jsnStorDbCfg.Db_passwd != nil {
			self.StorDBPass = *jsnStorDbCfg.Db_passwd
		}
		if jsnStorDbCfg.Max_open_conns != nil {
			self.StorDBMaxOpenConns = *jsnStorDbCfg.Max_open_conns
		}
		if jsnStorDbCfg.Max_idle_conns != nil {
			self.StorDBMaxIdleConns = *jsnStorDbCfg.Max_idle_conns
		}
	}

	if jsnGeneralCfg != nil {
		if jsnGeneralCfg.Dbdata_encoding != nil {
			self.DBDataEncoding = *jsnGeneralCfg.Dbdata_encoding
		}
		if jsnGeneralCfg.Default_reqtype != nil {
			self.DefaultReqType = *jsnGeneralCfg.Default_reqtype
		}
		if jsnGeneralCfg.Default_category != nil {
			self.DefaultCategory = *jsnGeneralCfg.Default_category
		}
		if jsnGeneralCfg.Default_tenant != nil {
			self.DefaultTenant = *jsnGeneralCfg.Default_tenant
		}
		if jsnGeneralCfg.Default_subject != nil {
			self.DefaultSubject = *jsnGeneralCfg.Default_subject
		}
		if jsnGeneralCfg.Connect_attempts != nil {
			self.ConnectAttempts = *jsnGeneralCfg.Connect_attempts
		}
		if jsnGeneralCfg.Reconnects != nil {
			self.Reconnects = *jsnGeneralCfg.Reconnects
		}
		if jsnGeneralCfg.Rounding_decimals != nil {
			self.RoundingDecimals = *jsnGeneralCfg.Rounding_decimals
		}
		if jsnGeneralCfg.Http_skip_tls_verify != nil {
			self.HttpSkipTlsVerify = *jsnGeneralCfg.Http_skip_tls_verify
		}
		if jsnGeneralCfg.Tpexport_dir != nil {
			self.TpExportPath = *jsnGeneralCfg.Tpexport_dir
		}
	}

	if jsnListenCfg != nil {
		if jsnListenCfg.Rpc_json != nil {
			self.RPCJSONListen = *jsnListenCfg.Rpc_json
		}
		if jsnListenCfg.Rpc_gob != nil {
			self.RPCGOBListen = *jsnListenCfg.Rpc_gob
		}
		if jsnListenCfg.Http != nil {
			self.HTTPListen = *jsnListenCfg.Http
		}
	}

	if jsnRaterCfg != nil {
		if jsnRaterCfg.Enabled != nil {
			self.RaterEnabled = *jsnRaterCfg.Enabled
		}
		if jsnRaterCfg.Balancer != nil {
			self.RaterBalancer = *jsnRaterCfg.Balancer
		}
		if jsnRaterCfg.Cdrstats != nil {
			self.RaterCdrStats = *jsnRaterCfg.Cdrstats
		}
		if jsnRaterCfg.Historys != nil {
			self.RaterHistoryServer = *jsnRaterCfg.Historys
		}
		if jsnRaterCfg.Pubsubs != nil {
			self.RaterPubSubServer = *jsnRaterCfg.Pubsubs
		}
		if jsnRaterCfg.Users != nil {
			self.RaterUserServer = *jsnRaterCfg.Users
		}
	}

	if jsnBalancerCfg != nil && jsnBalancerCfg.Enabled != nil {
		self.BalancerEnabled = *jsnBalancerCfg.Enabled
	}

	if jsnSchedCfg != nil && jsnSchedCfg.Enabled != nil {
		self.SchedulerEnabled = *jsnSchedCfg.Enabled
	}

	if jsnCdrsCfg != nil {
		if jsnCdrsCfg.Enabled != nil {
			self.CDRSEnabled = *jsnCdrsCfg.Enabled
		}
		if jsnCdrsCfg.Extra_fields != nil {
			if self.CDRSExtraFields, err = utils.ParseRSRFieldsFromSlice(*jsnCdrsCfg.Extra_fields); err != nil {
				return err
			}
		}
		if jsnCdrsCfg.Store_cdrs != nil {
			self.CDRSStoreCdrs = *jsnCdrsCfg.Store_cdrs
		}
		if jsnCdrsCfg.Rater != nil {
			self.CDRSRater = *jsnCdrsCfg.Rater
		}
		if jsnCdrsCfg.Cdrstats != nil {
			self.CDRSStats = *jsnCdrsCfg.Cdrstats
		}
		if jsnCdrsCfg.Reconnects != nil {
			self.CDRSReconnects = *jsnCdrsCfg.Reconnects
		}
		if jsnCdrsCfg.Cdr_replication != nil {
			self.CDRSCdrReplication = make([]*CdrReplicationCfg, len(*jsnCdrsCfg.Cdr_replication))
			for idx, rplJsonCfg := range *jsnCdrsCfg.Cdr_replication {
				self.CDRSCdrReplication[idx] = new(CdrReplicationCfg)
				if rplJsonCfg.Transport != nil {
					self.CDRSCdrReplication[idx].Transport = *rplJsonCfg.Transport
				}
				if rplJsonCfg.Server != nil {
					self.CDRSCdrReplication[idx].Server = *rplJsonCfg.Server
				}
				if rplJsonCfg.Synchronous != nil {
					self.CDRSCdrReplication[idx].Synchronous = *rplJsonCfg.Synchronous
				}
				if rplJsonCfg.Cdr_filter != nil {
					if self.CDRSCdrReplication[idx].CdrFilter, err = utils.ParseRSRFields(*rplJsonCfg.Cdr_filter, utils.INFIELD_SEP); err != nil {
						return err
					}
				}
			}
		}
	}

	if jsnCdrstatsCfg != nil {
		if jsnCdrstatsCfg.Enabled != nil {
			self.CDRStatsEnabled = *jsnCdrstatsCfg.Enabled
			if jsnCdrstatsCfg.Save_Interval != nil {
				if self.CDRStatsSaveInterval, err = utils.ParseDurationWithSecs(*jsnCdrstatsCfg.Save_Interval); err != nil {
					return err
				}
			}
		}
	}

	if jsnCdreCfg != nil {
		if self.CdreProfiles == nil {
			self.CdreProfiles = make(map[string]*CdreConfig)
		}
		for profileName, jsnCdre1Cfg := range jsnCdreCfg {
			if _, hasProfile := self.CdreProfiles[profileName]; !hasProfile { // New profile, create before loading from json
				self.CdreProfiles[profileName] = new(CdreConfig)
				if profileName != utils.META_DEFAULT {
					self.CdreProfiles[profileName] = self.dfltCdreProfile.Clone() // Clone default so we do not inherit pointers
				}
			}
			if err = self.CdreProfiles[profileName].loadFromJsonCfg(jsnCdre1Cfg); err != nil { // Update the existing profile with content from json config
				return err
			}
		}
	}

	if jsnCdrcCfg != nil {
		if self.CdrcProfiles == nil {
			self.CdrcProfiles = make(map[string]map[string]*CdrcConfig)
		}
		for profileName, jsnCrc1Cfg := range jsnCdrcCfg {
			if _, hasDir := self.CdrcProfiles[*jsnCrc1Cfg.Cdr_in_dir]; !hasDir {
				self.CdrcProfiles[*jsnCrc1Cfg.Cdr_in_dir] = make(map[string]*CdrcConfig)
			}
			if _, hasProfile := self.CdrcProfiles[profileName]; !hasProfile {
				if profileName == utils.META_DEFAULT {
					self.CdrcProfiles[*jsnCrc1Cfg.Cdr_in_dir][profileName] = new(CdrcConfig)
				} else {
					self.CdrcProfiles[*jsnCrc1Cfg.Cdr_in_dir][profileName] = self.dfltCdrcProfile.Clone() // Clone default so we do not inherit pointers
				}
			}
			if err = self.CdrcProfiles[*jsnCrc1Cfg.Cdr_in_dir][profileName].loadFromJsonCfg(jsnCrc1Cfg); err != nil {
				return err
			}
		}
	}

	if jsnSmFsCfg != nil {
		if err := self.SmFsConfig.loadFromJsonCfg(jsnSmFsCfg); err != nil {
			return err
		}
	}

	if jsnSmKamCfg != nil {
		if err := self.SmKamConfig.loadFromJsonCfg(jsnSmKamCfg); err != nil {
			return err
		}
	}

	if jsnSmOsipsCfg != nil {
		if err := self.SmOsipsConfig.loadFromJsonCfg(jsnSmOsipsCfg); err != nil {
			return err
		}
	}

	if jsnHistServCfg != nil {
		if jsnHistServCfg.Enabled != nil {
			self.HistoryServerEnabled = *jsnHistServCfg.Enabled
		}
		if jsnHistServCfg.History_dir != nil {
			self.HistoryDir = *jsnHistServCfg.History_dir
		}
		if jsnHistServCfg.Save_interval != nil {
			if self.HistorySaveInterval, err = utils.ParseDurationWithSecs(*jsnHistServCfg.Save_interval); err != nil {
				return err
			}
		}
	}

	if jsnPubSubServCfg != nil {
		if jsnPubSubServCfg.Enabled != nil {
			self.PubSubServerEnabled = *jsnPubSubServCfg.Enabled
		}
	}

	if jsnUserServCfg != nil {
		if jsnUserServCfg.Enabled != nil {
			self.UserServerEnabled = *jsnUserServCfg.Enabled
		}
		if jsnUserServCfg.Indexes != nil {
			self.UserServerIndexes = *jsnUserServCfg.Indexes
		}
	}

	if jsnMailerCfg != nil {
		if jsnMailerCfg.Server != nil {
			self.MailerServer = *jsnMailerCfg.Server
		}
		if jsnMailerCfg.Auth_user != nil {
			self.MailerAuthUser = *jsnMailerCfg.Auth_user
		}
		if jsnMailerCfg.Auth_passwd != nil {
			self.MailerAuthPass = *jsnMailerCfg.Auth_passwd
		}
		if jsnMailerCfg.From_address != nil {
			self.MailerFromAddr = *jsnMailerCfg.From_address
		}
	}
	return nil
}
