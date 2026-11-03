package timeutil

import "time"

var defaultTimeZone = time.FixedZone("Asia/Shanghai", 8*60*60)

func Now() time.Time {
	//todo 后续加偏移量
	return time.Now().In(defaultTimeZone)
}

func IsSameDay(ts1, ts2 int64) bool {
	t1 := time.UnixMilli(ts1).In(defaultTimeZone)
	t2 := time.UnixMilli(ts2).In(defaultTimeZone)

	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()

	return y1 == y2 && m1 == m2 && d1 == d2
}

// 判断是否为同一周（周一开头）
func IsSameWeek(ts1, ts2 int64) bool {
	return weekStart(
		time.UnixMilli(ts1).In(defaultTimeZone),
	).Equal(weekStart(
		time.UnixMilli(ts2).In(defaultTimeZone),
	))
}

// 判断是否为同一个自然月
func IsSameMonth(ts1, ts2 int64) bool {
	t1 := time.UnixMilli(ts1).In(defaultTimeZone)
	t2 := time.UnixMilli(ts2).In(defaultTimeZone)

	y1, m1, _ := t1.Date()
	y2, m2, _ := t2.Date()

	return y1 == y2 && m1 == m2
}

func weekStart(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}

	return time.Date(
		t.Year(), t.Month(), t.Day()-weekday+1,
		0, 0, 0, 0, t.Location(),
	)
}
