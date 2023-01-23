package synccalendar

import "time"

const DateFormat = "2006-01-02"

type Date struct {
	time.Time
}

func Today() Date {
	return NewDateFromTime(time.Now())
}

func NewDateFromTime(t time.Time) Date {
	return NewDate(t.Year(), t.Month(), t.Day(), t.Location())
}

func NewDate(year int, month time.Month, day int, loc *time.Location) Date {
	return Date{time.Date(year, month, day, 0, 0, 0, 0, loc)}
}

func (d Date) AddDate(years, months, days int) Date {
	t := d.Time.AddDate(years, months, days)
	return NewDate(t.Year(), t.Month(), t.Day(), t.Location())
}

func Parse(layout, value string) (Date, error) {
	t, err := time.Parse(layout, value)
	if err != nil {
		return Date{}, err
	}
	return NewDateFromTime(t), nil
}

func (d *Date) Set(v string) error {
	if d == nil {
		d = new(Date)
	}
	parsed, err := Parse(DateFormat, v)
	if err == nil {
		*d = parsed
	}
	return err
}

func (d Date) String() string {
	return d.Format(DateFormat)
}
