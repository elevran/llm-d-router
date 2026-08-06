# Topology Affinity Scorer Plugin

**Type:** `topology-affinity-scorer`

**Category:** `Affinity`

This plugin scores candidate endpoints by topology proximity to a peer endpoint selected
in an earlier scheduling phase.

## What it does

For a disaggregated prefill/decode request, `disagg-profile-handler` selects the decode
endpoint first, then runs the `prefill` profile to select a prefill endpoint. This plugin
runs in the `prefill` profile and grades each candidate by the tightest topology level it
shares with the decode pick (host, rack, zone, or region; same host implies same rack,
zone, and region), using a fixed proximity curve:

| Common level | Score | KV-transfer path        |
|--------------|-------|--------------------------|
| host         | 1.00  | NVLink                   |
| rack         | 0.20  | NIC, single switch       |
| zone         | 0.05  | multiple switch hops     |
| region       | 0.02  | inter-datacenter         |
| none         | 0.00  |                          |

The curve is shaped by KV-cache transfer bandwidth, not evenly spaced: same host
dominates same rack 5:1 and same zone 20:1, so co-location wins outright while the
looser levels still break ties among non-colocated candidates.

The curve is hardcoded, not configurable per level, in this release. A missing value
never matches, including empty against empty: an endpoint with no `Hostname` never scores
`host` proximity against a peer that also has no `Hostname`.

Every candidate scores 0 when no peer topology is available (the peer endpoint is
unknown, or has no non-empty topology field) or when the candidate is missing the
`Topology` attribute. A zero score contributes nothing to the weighted sum rather than
skewing it.

## Inputs consumed

Reads the `Topology` attribute (`topology-extractor`) from the candidate endpoints and
from the peer endpoint. The peer endpoint is resolved from, in order:

1. The `peer-endpoint` request attribute, published by `disagg-profile-handler` before
   running the `prefill` profile in single-EPP deployments.
2. The `peerTopologyHeader` request header, set by the prefill-side response stamper in
   coordinator deployments with separate prefill and decode EPPs. Not yet implemented;
   configuring this parameter has no effect until that stamper lands.

## Configuration

| Parameter               | Required | Default | Description                                                                  |
|--------------------------|----------|---------|--------------------------------------------------------------------------------|
| `topologyProducerName`   | no       | default producer | `topology-extractor` instance to read the `Topology` attribute from. |
| `peerTopologyHeader`     | no       | unset   | Header carrying the peer topology when the peer endpoint is not in-process. Not yet implemented. |

**Configuration Example:**
```yaml
plugins:
  - type: topology-extractor
  - type: topology-affinity-scorer
    name: prefill-topology-affinity
schedulingProfiles:
  - name: prefill
    plugins:
      - pluginRef: prefill-topology-affinity
        weight: 1
```

## See also

The `topology-affinity-filter` plugin drops candidates below a minimum affinity instead
of scoring them, and is the filtering counterpart of this scorer.
