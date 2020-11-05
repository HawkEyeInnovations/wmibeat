# WMIbeat

Welcome to WMIbeat.  WMIbeat is a [beat](https://github.com/elastic/beats) that allows you to run arbitrary WMI queries
and index the results into [elasticsearch](https://github.com/elastic/elasticsearch) so you can monitor Windows machines.

Ensure that this folder is at the following location:
`${GOPATH}/github.com/eskibars`

## Getting Started with WMIbeat
To get running with WMIbeat, run "go build" and then run wmibeat.exe, as in the below `run` section.
If you don't want to build your own, hop over to the "releases" page to download the latest.

### Configuring
To configure the WMI queries to run, you need to change wmibeat.yml.  Working from the default example:

    queries:
    - class: Win32_OperatingSystem
      period: 1m
      fields:
      - name: FreePhysicalMemory
      - name: FreeSpaceInPagingFiles
      - name: FreeVirtualMemory
      - name: NumberOfProcesses
      - name: NumberOfUsers
    - class: Win32_PerfFormattedData_PerfDisk_LogicalDisk
      period: 10s
      fields:
      - name: Name
      - name: FreeMegabytes
      - name: PercentFreeSpace
      - name: CurrentDiskQueueLength
      - name: DiskReadsPerSec
      - name: DiskWritesPerSec
      - name: DiskBytesPerSec
      - name: PercentDiskReadTime
      - name: PercentDiskWriteTime
      - name: PercentDiskTime
      whereclause: Name != "_Total"
    - class: Win32_PerfFormattedData_PerfOS_Memory
      namespace: root/CIMV2
      period: 10s
      fields:
      - name: CommittedBytes
      - name: AvailableBytes
      - name: PercentCommittedBytesInUse
    - class: Win32_PerfFormattedData_Counters_ProcessorInformation
      period: 10s
      fields:
        - name: Name
        - name: PercentProcessorTime
          int: true
        - name: PercentProcessorUtility
          int: true
      whereclause: Name Like "%,_Total"

For each class we can define: the wmi namespace, the polling period, the fields and the whereclause. Fields can have the optional boolean argument of `int` which if set to true will convert the string value returned from WMI to an integer value. This is useful for `Win32_PerfFormattedData_Counters_ProcessorInformation` where `PercentProcessorTime` and `PercentProcessorUtility` are returned as strings from WMI but are actually numerical values.

If there are multiple results, for any WMI class, WMIbeat will add the results as arrays.  If you need some help with what classes/fields, you can try [WMI Explorer](https://wmie.codeplex.com/).
Note that many of the more interesting classes are "Perf" classes, which has a special checkbox to see in that tool.
	  
### Run

To run WMIbeat with debugging output enabled, run:

```
./wmibeat -c wmibeat.yml -e -d "*"
```

## Build your own Beat
Beats is open source and has a convenient Beat generator, from which this project is based.
For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).
