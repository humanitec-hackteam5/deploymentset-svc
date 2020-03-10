# Data Format

Deployment set has the following JSON format:
```
{
    "modules": {
        "module_1_release": {
            "helmchart": "<helm chart for module_1>",
            "values": {
                // Specific values for module_1 helm chart
            }
        },
        "module_2_release": {
            "helmchart": "<helm chart for module_2>",
            "values": {
                // Specific values for module_2 helm chart
            }
        }
    }
}
```

Specific Helm chart values are defined in Humanitec Helm charts repository: 
https://github.com/Humanitec/walhall-helm-charts