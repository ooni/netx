#!/usr/bin/env python3
import json, sys
connids, txids = {}, {}
for line in sys.stdin:
    ####print('> ', line)
    d = json.loads(line)
    if 'Connect' in d:
        k = d['Connect']['LocalAddress'] + d['Connect']['RemoteAddress']
        connids[k] = d['Connect']['ConnID']
    elif 'HTTPConnectionReady' in d:
        v = d['HTTPConnectionReady']['LocalAddress'] + d['HTTPConnectionReady']['RemoteAddress']
        txids[d['HTTPConnectionReady']['TransactionID']] = v
    for k in d:
        if k.startswith('HTTP'):
            v = txids[d[k]['TransactionID']]
            connid = connids[v]
            d[k]['ConnID'] = connid
    json.dump(d, sys.stdout, indent=4, sort_keys=True)
    sys.stdout.write("\n\n")

