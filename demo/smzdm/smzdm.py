#!/usr/bin/env python
# coding: utf-8

import requests
import time
import json
import urllib2
import urllib


config = {
    'min_people': 3,
    'min_worthy' : 0.5,
}

def send_msg(to, content):
    wechat_api = 'http://localhost:8765/send'
    requests.get(wechat_api, params={'to': to, 'content': content})


def get_real_time_data():
    ctime = int(time.time())
    headers = {
        'Accept': 'application/json, text/javascript, */*; q=0.01',
        'Host': 'www.smzdm.com',
        'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_6) AppleWebKit/603.2.5 (KHTML, like Gecko) Version/10.1.1 Safari/603.2.5'
    }

    url = 'http://www.smzdm.com/json_more?timesort=' + str(ctime)
    r = requests.get(url=url, headers=headers)

    data = r.text
    return data

def worth(data):
    data = json.loads(data)
    global config
    for i in data:
        try:
            worthy = (float)(i['article_worthy'])
            unworthy = (float)(i['article_unworthy'])
            total = worthy + unworthy
            if total >= config['min_people'] \
                and worthy / total >= config['min_worthy'] \
                and i["article_referrals"] != u"商家自荐":
                content = '%d%% (%d:%d), %s, %s, %s' % (worthy / total * 100, worthy, total, i['article_price'], i['article_title'], i['article_url'])
                print content
                send_msg(u'xxxxx', content)
        except Exception as e:
            pass

if __name__ == '__main__':
    worth(get_real_time_data())
