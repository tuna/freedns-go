#! /usr/bin/env python3

import requests

IP_LIST_URL = "https://raw.githubusercontent.com/17mon/china_ip_list/master/china_ip_list.txt"

def cidr_list():
    r = requests.get(IP_LIST_URL)
    if r.status_code != 200:
        raise Exception("%s status code is %d" % (IP_LIST_URL, r.status_code))
        
    return r.text.split()

def to_int(a, b, c, d):
    return a*256*256*256 + b*256*256 + c*256 + d

def parse(cidr):
    ip, mask = cidr.split("/")
    mask = int(mask)
    a, b, c, d = list(map(int, ip.split(".")))
    i = to_int(a, b, c, d)
    start = i >> (32 - mask) << (32 - mask)
    end = start + 2**(32 - mask) - 1
    return start, end

def gen():
    s = """package chinaip

var chinaIPs = [][]uint32{
"""
    cidrs = cidr_list()
    for cidr in cidrs:
        if (len(cidr) > 3):
            start, end = parse(cidr)
        s += "	{%d, %d},     // %s\n" % (start, end, cidr)

    s += "}\n"
    return s

def main():
    s = gen()
    with open("db.go", "w") as f:
        f.write(s)

if __name__ == "__main__":
    main()
        