package mibs

import (
	"os"
	"strings"

	"github.com/kentik/ktranslate/pkg/eggs/logger"
	"github.com/kentik/ktranslate/pkg/kt"

	"gopkg.in/yaml.v2"
)

type Trap struct {
	Oid           string `yaml:"trap_oid"`
	Name          string `yaml:"trap_name"`
	DropUndefined bool   `yaml:"drop_undefined"`
	Events        []OID  `yaml:"events"`
}

type TrapBase struct {
	logger.ContextL `yaml:"-"`
	Traps           []Trap `yaml:"traps"`
	From            string `yaml:"from,omitempty"`
}

func (t *Trap) DropUndefinedVars() bool {
	if t == nil {
		return false
	}
	return t.DropUndefined
}

func normalizeOid(oid string) string {
	if strings.HasPrefix(oid, ".") {
		return oid
	}
	return "." + oid
}

func (mdb *MibDB) parseTrapsFromYml(fname string, file os.DirEntry, extends map[string]*Profile) error {
	t := TrapBase{ContextL: mdb.log, From: file.Name()}
	data, err := os.ReadFile(fname)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &t)
	if err != nil {
		return err
	}

	added := 0
	for _, trap := range t.Traps {
		for _, event := range trap.Events {
			mib := &kt.Mib{
				Oid:        event.Oid,
				Name:       event.Name,
				Enum:       event.Enum,
				Tag:        event.Tag,
				Conversion: event.Conversion,
				Extra:      trap.Name,
				Mib:        trap.Oid,
			}
			if len(mib.Enum) > 0 {
				mib.EnumRev = make(map[int64]string)
			}
			for k, v := range mib.Enum {
				mib.Enum[strings.ToLower(k)] = v
				mib.EnumRev[v] = k
			}
			added++
			mdb.trapMibs[normalizeOid(mib.Oid)] = mib
		}
		mdb.traps[normalizeOid(trap.Oid)] = trap
	}

	mdb.log.Infof("Loading %d snmp trap data points and %d traps from %s.", added, len(mdb.traps), fname)

	return nil
}
