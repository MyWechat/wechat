#!/usr/bin/env python
# coding: utf-8

import icalendar
import urllib2
import urllib
from dateutil.rrule import *
from datetime import *
from pprint import pprint
from pytz import timezone

def get_calendar_content():
    # ios 下的 webcal 直接修改为 https 即可
    # url = 'https://p21-calendars.icloud.com/published/2/P4C8TJk-btfRJ8u5eSxmMhfyUlaE4SFHKm-x_yHgdzxdga11W80B-hwJ6GR7ZjN75GQ6gctlxgdc4LY2khFXhA6nBEYLKO8ZGUtaugENEVI'
    # return urllib2.urlopen(url).read()
    with open('demo.ical', 'r') as cal:
        return cal.read()

def send_msg(to, content):
    to = to.encode('utf8')
    content = content.encode('utf8')
    wechat_api = 'http://localhost:8765/send?'+urllib.urlencode({'to': to, 'content': content})
    urllib2.urlopen(wechat_api).read()

def main():
    content = get_calendar_content()
    cal = icalendar.Calendar.from_ical(content)

    notifications = []

    for event in cal.walk('vevent'):
        rules_text = '\n'.join([line for line in event.content_lines() if line.startswith('RRULE')])
        rules = rruleset()
        first_rule = rrulestr(rules_text, dtstart=event.get('dtstart').dt)
        rules.rrule(first_rule)
        now = datetime.now(tz=timezone('Asia/Shanghai'))
        in_today = (now + timedelta(days=1)).replace(hour=0, minute=0, second=0, microsecond=0)
        rules_today = rules.between(now, in_today)

        if len(rules_today) > 0:
            notifications.append('%s %s' % (event.get('summary'), ' / '.join([str(x) for x in rules_today])))

    send_msg(u'xxxx', u'今天的日历提醒有: '+' '.join(notifications))



if __name__ == '__main__':
    main()