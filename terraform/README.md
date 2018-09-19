# Usage

```
export TF_REGION=ap-jp-1 
export TF_USER=d038720 
export TF_PASSWORD=$(security find-generic-password -a $USER -s openstack -w) 

# env TF_REGION=ap-jp-1 TF_USER=d038720 TF_PASSWORD=(security find-generic-password -a $USER -s openstack -w) make plan

make init
make plan 
make apply
```
