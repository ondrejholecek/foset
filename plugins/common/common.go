// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

package plugin_common

import (
	"foset/fortisession"
	"foset/iproviders"
)

// Hooks specify the plugin callbacks.
// Hooks BeforeFilter and AfterFilter expect pointer to Session, which they can fully
// access (however, only the `Custom` fieldentries should be added by plugin). 
// Their return boolean specifies
// whether `foset` should ignore the session (true = to not display it) or no (false).
type Hooks struct {
	Start            func()
	BeforeFilter     func(*fortisession.Session)(bool)
	AfterFilter      func(*fortisession.Session)(bool)
	Finished         func()
}

// FosetPlugin describes the plugin and specify its usage.
type FosetPlugin struct {
	Hooks   Hooks

	// Parameters filled by main part of Foset for use inside of plugins
	Version   string
	Commit    string
	Filename  string
	Filter    string

	Inputs    *iproviders.IProviders
}


