package festival

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/nosixtools/solarlunar"
)

var RULE_PATTERN = "^(solar|lunar)\\((?:m(\\d+)):(ld|(?:d|(?:fw|lw|w(\\d+))n|(?:s\\d+))(\\d+))\\)=.+$"
var PATTERN = "^(solar|lunar)\\((?:m(\\d+)):(ld|(?:d|(?:fw|lw|w(\\d+))n|(?:s\\d+))(\\d+))\\)$"
var MONTH_SOLAR_FESTIVAL = map[string][]string{}
var MONTH_LUNAR_FESTIVAL = map[string][]string{}
var SOLAR = "solar"
var LUNAR = "lunar"
var DATELAYOUT = "2006-01-02"

type Festival struct {
	filename string
	local    string
}

func NewFestival(filename, local string) *Festival {
	if filename == "" {
		filename = "./festival.json"
	}
	if local == "" {
		local = "Local"
	}
	readFestivalRuleFromFile(filename)
	return &Festival{
		filename: filename,
		local:    local,
	}
}

func (f *Festival) GetFestivals(solarDay string) (festivals []string) {
	festivals = []string{}
	loc, _ := time.LoadLocation(f.local)

	//处理公历节日
	tempDate, _ := time.ParseInLocation(DATELAYOUT, solarDay, loc)
	for _, festival := range processRule(tempDate, MONTH_SOLAR_FESTIVAL, false, solarDay) {
		festivals = append(festivals, festival)
	}
	//处理农历节日
	lunarDate, isLeapMonth := solarlunar.SolarToLuanr(solarDay)
	if !isLeapMonth {
		// 此方式解析农历可能会失败
		tempDate, err := time.ParseInLocation(DATELAYOUT, lunarDate, loc)
		if err != nil {
			items := strings.Split(lunarDate, "-")
			if len(items) != 3 {
				return
			}

			y, yerr := strconv.Atoi(items[0])
			m, merr := strconv.Atoi(items[0])
			d, derr := strconv.Atoi(items[0])
			if yerr != nil || merr != nil || derr != nil {
				return
			}
			tempDate = time.Date(y, time.Month(m), d, 0, 0, 0, 1, loc)
		}
		for _, festival := range processRule(tempDate, MONTH_LUNAR_FESTIVAL, true, solarDay) {
			festivals = append(festivals, festival)
		}
	}
	return
}

func readFestivalRuleFromFile(filename string) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	rules, err := simplejson.NewJson(bytes)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	solarData := rules.Get(SOLAR)
	if solarData != nil {
		solarMap, err := solarData.Map()
		if err != nil {
			fmt.Println(err.Error())
		}
		for key, value := range solarMap {
			for _, item := range value.([]interface{}) {
				v := item.(string)
				is, err := regexp.MatchString(RULE_PATTERN, v)
				if err != nil {
					fmt.Println(err.Error())
				}
				if is {
					if _, ok := MONTH_SOLAR_FESTIVAL[key]; ok {
						MONTH_SOLAR_FESTIVAL[key] = append(MONTH_SOLAR_FESTIVAL[key], v)
					} else {
						temp := []string{v}
						MONTH_SOLAR_FESTIVAL[key] = temp
					}
				}
			}
		}
	}
	lunarData := rules.Get(LUNAR)
	if lunarData != nil {
		lunarMap, err := lunarData.Map()
		if err != nil {
			fmt.Println(err.Error())
		}
		for key, value := range lunarMap {
			for _, item := range value.([]interface{}) {
				v := item.(string)
				is, err := regexp.MatchString(RULE_PATTERN, v)
				if err != nil {
					fmt.Println(err.Error())
				}
				if is {
					if _, ok := MONTH_LUNAR_FESTIVAL[key]; ok {
						MONTH_LUNAR_FESTIVAL[key] = append(MONTH_LUNAR_FESTIVAL[key], v)
					} else {
						temp := []string{v}
						MONTH_LUNAR_FESTIVAL[key] = temp
					}
				}
			}
		}
	}
}

func processRule(date time.Time, ruleMap map[string][]string, isLunar bool, solarDay string) []string {
	festivals := []string{}
	year := int(date.Year())
	month := strconv.Itoa(int(date.Month()))
	day := strconv.Itoa(date.Day())
	rules := ruleMap[month]
	for _, rule := range rules {
		items := strings.Split(rule, "=")
		reg, _ := regexp.Compile(PATTERN)
		subMatch := reg.FindStringSubmatch(items[0])
		festivalMonth := subMatch[2]
		if strings.HasPrefix(subMatch[3], "s456") && !isLunar { //特殊处理清明节
			festivalDay := getQingMingFestival(year)
			if month == festivalMonth && day == festivalDay {
				festivals = append(festivals, items[1])
			}
			continue
		} else if strings.HasPrefix(subMatch[3], "s345") && !isLunar { //特殊处理寒食节，为清明节前一天
			festivalDay := getQingMingFestival(year)
			intValue, err := strconv.Atoi(festivalDay)
			if err != nil {
				fmt.Print(err.Error())
				continue
			}
			festivalDay = strconv.Itoa(intValue - 1)
			if month == festivalMonth && day == festivalDay {
				festivals = append(festivals, items[1])
			}
		} else if strings.HasPrefix(subMatch[3], "d") {
			festivalDay := subMatch[5]
			if month == festivalMonth && day == festivalDay {
				festivals = append(festivals, items[1])
			}
			continue
		} else if strings.HasPrefix(subMatch[3], "w") {
			festivalWeek := subMatch[3][1:2]
			festivalDayOfWeek := subMatch[3][3:4]
			festivalWeekInt, err := strconv.Atoi(festivalWeek)
			if err != nil {
				continue
			}
			festivalDayOfWeekInt, err := strconv.Atoi(festivalDayOfWeek)
			if err != nil {
				continue
			}

			if IsWeekdayN(date, time.Weekday(festivalDayOfWeekInt-1), festivalWeekInt) {
				festivals = append(festivals, items[1])
			}
			continue
		} else if strings.HasPrefix(subMatch[3], "lw") {
			festivalDayOfWeek, _ := strconv.Atoi(subMatch[3][3:4])
			if IsWeekdayN(date, time.Weekday(festivalDayOfWeek-1), -1) {
				// if isDayOfLastWeeekInTheMonth(date, festivalDayOfWeek) {
				festivals = append(festivals, items[1])
			}
			continue
		} else if strings.HasPrefix(subMatch[3], "ld") && isLunar { //特殊处理除夕节日
			if month == "12" && day == "29" {
				nextLunarDay := lunarDateAddOneDay(solarDay)
				newMonth := strconv.Itoa(int(nextLunarDay.Month()))
				if month != newMonth {
					festivals = append(festivals, items[1])
				}
			} else if month == "12" && day == "30" {
				festivals = append(festivals, items[1])
			}
			continue
		}
	}
	return festivals
}

// 清明节算法 公式：int((yy*d+c)-(yy/4.0)) 公式解读：y=年数后2位，d=0.2422，1=闰年数，21世纪c=4081，20世纪c=5.59
func getQingMingFestival(year int) string {
	var val float64
	if year >= 2000 { //21世纪
		val = 4.81
	} else { //20世纪
		val = 5.59
	}
	d := float64(year % 100)
	day := int(d*0.2422 + val - float64(int(d)/4))
	return strconv.Itoa(day)
}

func lunarDateAddOneDay(solarDay string) time.Time {
	tempDate, err := time.Parse(DATELAYOUT, solarDay)
	if err != nil {
		fmt.Println(err.Error())
	}
	dayDuaration, _ := time.ParseDuration("24h")
	nextDate := tempDate.Add(dayDuaration)
	lunarDate, _ := solarlunar.SolarToLuanr(nextDate.Format(DATELAYOUT))
	nexLunarDay, err := time.Parse(DATELAYOUT, lunarDate)
	if err != nil {
		fmt.Println(err.Error())
	}
	return nexLunarDay
}

// IsWeekdayN reports whether the given date is the nth occurrence of the
// day in the month.
//
// The value of n affects the direction of counting:
//   n > 0: counting begins at the first day of the month.
//   n == 0: the result is always false.
//   n < 0: counting begins at the end of the month.
func IsWeekdayN(date time.Time, day time.Weekday, n int) bool {
	cday := date.Weekday()
	if cday != day || n == 0 {
		return false
	}

	if n > 0 {
		return (date.Day()-1)/7 == (n - 1)
	}

	n = -n
	last := time.Date(date.Year(), date.Month()+1,
		1, 12, 0, 0, 0, date.Location())
	lastCount := 0
	for {
		last = last.AddDate(0, 0, -1)
		if last.Weekday() == day {
			lastCount++
		}
		if lastCount == n || last.Month() != date.Month() {
			break
		}
	}
	return lastCount == n && last.Month() == date.Month() &&
		last.Day() == date.Day()

}

func isLeapYear(year int) bool {
	if year%4 == 0 && year%100 != 0 || year%400 == 0 {
		return true
	}
	return false
}

func isDayOfLastWeeekInTheMonth(now time.Time, weekNumber int) bool {
	var endDayOfMonth time.Time
	year := now.Year()
	month := int(now.Month())
	isLeap := isLeapYear(year)
	if month == 2 {
		if isLeap {
			endDayOfMonth = time.Date(now.Year(), now.Month(), 29, 23, 59, 59, 1, time.UTC)
		} else {
			endDayOfMonth = time.Date(now.Year(), now.Month(), 28, 23, 59, 59, 1, time.UTC)
		}
	} else if month == 1 || month == 3 || month == 5 || month == 7 || month == 8 || month == 10 || month == 12 {
		endDayOfMonth = time.Date(now.Year(), now.Month(), 31, 23, 59, 59, 1, time.UTC)
	} else {
		endDayOfMonth = time.Date(now.Year(), now.Month(), 30, 23, 59, 59, 1, time.UTC)
	}
	_, lastWeekOfMonth := endDayOfMonth.ISOWeek()
	_, nowWeekOfMonth := now.ISOWeek()
	dayOfWeek := (int(endDayOfMonth.Weekday()) + 1) % 7
	if dayOfWeek < weekNumber && lastWeekOfMonth > nowWeekOfMonth {
		dayDuaration, _ := time.ParseDuration("-24h")
		endDayOfMonth = endDayOfMonth.Add(dayDuaration * time.Duration(7))
		_, lastWeekOfMonth = endDayOfMonth.ISOWeek()
	}
	if lastWeekOfMonth == nowWeekOfMonth {
		nowDayOfWeek := (int(now.Weekday()) + 1) % 7
		if nowDayOfWeek == weekNumber {
			return true
		}
	}
	return false
}
