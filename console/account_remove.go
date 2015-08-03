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

package console

import "github.com/cgrates/cgrates/utils"

func init() {
	c := &CmdRemoveAccount{
		name:      "account_remove",
		rpcMethod: "ApierV1.RemoveAccount",
	}
	commands[c.Name()] = c
	c.CommandExecuter = &CommandExecuter{c}
}

// Commander implementation
type CmdRemoveAccount struct {
	name      string
	rpcMethod string
	rpcParams *utils.AttrRemoveAccount
	*CommandExecuter
}

func (self *CmdRemoveAccount) Name() string {
	return self.name
}

func (self *CmdRemoveAccount) RpcMethod() string {
	return self.rpcMethod
}

func (self *CmdRemoveAccount) RpcParams(reset bool) interface{} {
	if reset || self.rpcParams == nil {
		self.rpcParams = &utils.AttrRemoveAccount{Direction: utils.OUT}
	}
	return self.rpcParams
}

func (self *CmdRemoveAccount) PostprocessRpcParams() error {
	return nil
}

func (self *CmdRemoveAccount) RpcResult() interface{} {
	var s string
	return &s
}
