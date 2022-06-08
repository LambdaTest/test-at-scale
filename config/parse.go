package config

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/spf13/viper"
)

const tagPrefix = "viper"

// populateNucleusConfig is used to parse config read through viper
func populateNucleusConfig(config *NucleusConfig) (*NucleusConfig, error) {
	err := recursivelySet(reflect.ValueOf(config), "")
	if err != nil {
		return nil, err
	}

	return config, nil
}

// populateSynapseConfig is used to parse config read through viper
func populateSynapseConfig(config *SynapseConfig) (*SynapseConfig, error) {
	err := recursivelySet(reflect.ValueOf(config), "")
	if err != nil {
		return nil, err
	}

	return config, nil
}

// recursivelySet is used to recursively set conf read from
// files to golang structs. Since nested values are accessed using periods
// we need to recursively parse the values
func recursivelySet(val reflect.Value, prefix string) error {
	if val.Kind() != reflect.Ptr {
		return errors.New("WTF")
	}

	// dereference
	val = reflect.Indirect(val)
	if val.Kind() != reflect.Struct {
		return errors.New("FML")
	}

	// grab the type for this instance
	vType := reflect.TypeOf(val.Interface())

	// go through child fields
	for i := 0; i < val.NumField(); i++ {
		thisField := val.Field(i)
		thisType := vType.Field(i)
		tags := getTags(thisType)
		// try to fetch value for each key using multiple tags
		for _, tag := range tags {
			key := prefix + tag
			switch thisField.Kind() {
			case reflect.Struct:
				if err := recursivelySet(thisField.Addr(), key+"."); err != nil {
					return err
				}
			case reflect.Int:
				fallthrough
			case reflect.Int32:
				fallthrough
			case reflect.Int64:
				// you can only set with an int64 -> int
				configVal := int64(viper.GetInt(key))
				// skip the update if tag is not set in viper
				if viper.GetInt(key) == 0 && thisField.Int() != 0 {
					continue
				}
				thisField.SetInt(configVal)
			case reflect.String:
				// skip the update if tag is not set in viper
				if viper.GetString(key) == "" && thisField.String() != "" {
					continue
				}
				thisField.SetString(viper.GetString(key))
			case reflect.Bool:
				// skip the update if tag is not set in viper
				if !viper.GetBool(key) && thisField.Bool() {
					continue
				}
				thisField.SetBool(viper.GetBool(key))
			case reflect.Map:
				continue
			default:
				return fmt.Errorf("unexpected type detected ~ aborting: %s", thisField.Kind())
			}
		}
	}

	return nil
}

func getTags(field reflect.StructField) []string {
	// check if maybe we have a special magic tag
	tag := field.Tag
	values := []string{}
	if tag != "" {
		for _, prefix := range []string{tagPrefix, "yaml", "json", "env", "mapstructure"} {
			if v := tag.Get(prefix); v != "" {
				values = append(values, v)
			}
		}
		return values
	}

	return []string{field.Name}
}
