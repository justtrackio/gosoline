# Unique Id

The unique id is based on the Twitter snowflake id / Sonyflake Go Variation. It combines a unix timestamp, a machine id
and a sequence id to generate unique, numeric 64bit ids. These ids are k-sortable which means that they can be arranged
into sortable batches of size k. Within one batch there can be no guarantee for ordering. The k-factor in this case
depends on the timestamp. For example, if there are multiple ids generated on the same timestamp but by different
instances, they only will be sortable within the timestamp batch. Check here for further
info: https://github.com/sony/sonyflake

# Example

1. Upon EC2 instance start, add a uniqueid api server to your cluster with a predefined port. Use the `api.port` setting
   here. Leave the `machine_id` setting empty such that the instance IP will be used instead
2. In your code, request the `unique_id.Generator` by calling `unique_id.ProvideGenerator` method and set the predefined
   port
3. You can access the method call `NextId() (*int64, error)`. It will do a request to the local api and fetch the new
   request
