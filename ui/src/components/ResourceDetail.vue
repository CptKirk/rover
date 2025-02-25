<template>
  <fieldset id="resource-details">
    <legend>Details</legend>
    <div class="resource-detail">
      <div v-if="!resourceID">
        <span>Please select a resource on your right.</span>
      </div>
      <div v-else>
        <dd class="key">{{ primitiveType }}</dd>
        <span
          class="tag is-small resource-action"
          v-if="resourceChange.action"
          >{{ resourceChange.action }}</span
        >
        <dt class="value resource-id">
          {{ resource.id }}
          <button
            class="copy-button"
            @click="copyText(resource.id, 'rid')"
            ref="rid"
          >
            Copy
          </button>
        </dt>

        <!-- <dd class="key">Resource Type</dd>
        <dt class="value">{{ resource.resource_type }}</dt>
c
        <dd class="key">Resource Name</dd>
        <dt class="value">{{ resource.resource_name }}</dt> -->

        <nav class="tabs is-full">
          <a
            @click="selectTab('config')"
            :class="{ active: curTab === 'config' }"
            >Config</a
          >
          <a
            @click="selectTab('current')"
            :class="{ active: curTab === 'current', disabled: hasNoState }"
            >Current State</a
          >
          <a
            @click="selectTab('proposed')"
            :class="{ active: curTab === 'proposed', disabled: hasNoState }"
            >Proposed State</a
          >
          <a
            @click="selectTab('diff')"
            :class="{ active: curTab === 'diff', disabled: hasNoState }"
            >State diff</a
          >
        </nav>

        <div class="tab-container" v-if="curTab === 'config'">
          <!-- {{ resourceConfig }} -->
          <span
            v-if="
              resourceConfig.isChild == 'rover-for-each-child-resource-true'
            "
            class="is-child-resource"
            >Please check parent resource</span
          >
          <div v-for="(val, k) in resourceConfig" :key="k" v-else>
            <dd class="key">{{ k }}</dd>
            <dt class="value" v-if="val">
              {{ getConfigValue(val) }}
              <button
                class="copy-button"
                @click="copyText(getStringConfigValue(val), `${resource.id}-${k}`)"
                :ref="`${resource.id}-${k}`"
              >
                Copy
              </button>
            </dt>
            <dt class="value" v-else>null</dt>
          </div>
        </div>

        <div class="tab-container" v-if="curTab === 'current'">
          <span v-if="resourceChange.before">
            <div v-for="(val, k) in resourceChange.before" :key="k">
              <dd class="key">{{ k }}</dd>
              <dt class="value" v-if="val">
                {{ getBeforeValue(val) }}
                <button
                  class="copy-button"
                  @click="copyText(getStringBeforeValue(val), `${resource.id}-${k}`)"
                  :ref="`${resource.id}-${k}`"
                >
                  Copy
                </button>
              </dt>
              <dt class="value" v-else>null</dt>
            </div>
          </span>
          <span v-else>Resource doesn't currently exist.</span>
        </div>

        <div class="tab-container" v-if="curTab === 'proposed'">
          <!-- {{ resourceChange }} -->

          <div v-for="(val, k) in resourceChange.after" :key="k">
            <dd class="key">{{ k }}</dd>
            <dt
              class="value"
              v-if="val"
              :class="{ 'unknown-value': val.unknown }"
            >
              {{ val.unknown ? "Value Unknown" : val }}
              <button
                class="copy-button"
                @click="copyText(getStringBeforeValue(val), `${resource.id}-${k}`)"
                :ref="`${resource.id}-${k}`"
              >
                Copy
              </button>
            </dt>
            <dt class="value" v-else>null</dt>
          </div>
        </div>

        <div class="tab-container" v-if="curTab === 'diff'">
          <!-- {{ resourceChange }} -->

          <div v-for="(val, k) in resourceChange.after" :key="k">
            <div v-if="((!lodashIsEqual(resourceChange.before[k], val)) && (resourceChange.before[k] !== null && val !== null))">
              <dd class="key">{{ k }}</dd>
              <dt
                class="value-before"
                v-if="resourceChange.before[k]"
                :class="{ 'unknown-value': resourceChange.before[k].unknown }"
              >
                {{ resourceChange.before[k].unknown ? "Value Unknown" : resourceChange.before[k] }}
                <button
                  class="copy-button"
                  @click="copyText(getStringBeforeValue(resourceChange.before[k]), `${resource.id}-${k}`)"
                  :ref="`${resource.id}-${k}`"
                >
                  Copy
                </button>
              </dt>
              <dt class="value-before" v-else>null</dt>

              <dt
                class="value-after"
                v-if="val"
                :class="{ 'unknown-value': val.unknown }"
              >
                {{ val.unknown ? "Value Unknown" : val }}
                <button
                  class="copy-button"
                  @click="copyText(getStringBeforeValue(val), `${resource.id}-${k}`)"
                  :ref="`${resource.id}-${k}`"
                >
                  Copy
                </button>
              </dt>
              <dt class="value-after" v-else>null</dt>
            </div>
          </div>
        </div>
      </div>
    </div>
  </fieldset>
</template>

<script>
import axios from "axios";
import copy from "copy-to-clipboard";
import _ from 'lodash';

export default {
  name: "ResourceDetail",
  props: {
    resourceID: String,
  },
  data() {
    return {
      curTab: "config",
      overview: {},
    };
  },
  methods: {
    selectTab(tab) {
      if (!this.hasNoState) {
        this.curTab = tab;
      }
    },
    copyText(text, ref) {
      copy(text, {
        onCopy: this.updateCopyText(ref),
      });
    },
    lodashIsEqual(val1, val2) {
      return _.isEqual(val1, val2)
    },
    updateCopyText(ref) {
      // Use the first element if returns an array
      if (Array.isArray(this.$refs[ref])) {
        this.$refs[ref][0].innerText = "Copied";
        setTimeout(() => {
          this.$refs[ref][0].innerText = "Copy";
        }, 1000);
      } else {
        this.$refs[ref].innerText = "Copied";
        setTimeout(() => {
          this.$refs[ref].innerText = "Copy";
        }, 1000);
      }
    },
    getStringConfigValue(val) {
      if (val.references) {
        return val.references.join(", ");
      } else if (val.constant_value) {
        return val.constant_value;
      } else {
        return val ? JSON.stringify(val) : "null";
      }
    },
    getConfigValue(val) {
      if (val.references) {
        return val.references.join(", ");
      } else if (val.constant_value) {
        return val.constant_value;
      } else {
        return val ? val : "null";
      }
    },
    getStringBeforeValue(val) {
      return val ? JSON.stringify(val) : "null";
    },
    getBeforeValue(val) {
      return val ? val : "null";
    },
    getAfterValue(val) {
      return val ? JSON.stringify(val) : "null";
    },
    getResourceConfig(resourceID, model, isChild) {
      let configID = model.states[resourceID]?.config_id ? model.states[resourceID]?.config_id : resourceID.replace(/\[[^[\]]*\]/g, "");

      let config;

      if (isChild) return { isChild: "rover-for-each-child-resource-true" };

      // If variable, return variable config
      if ((config = model.configs[configID]?.variable_config) !== undefined) {
        return config;
      }

      // If output, return output config
      if ((config = model.configs[configID]?.output_config) !== undefined) {
        return config;
      }

      // If module, return module config
      if ((config = model.configs[configID]?.module_config) !== undefined) {
        return config;
      }

      if ((config = model.configs[configID]?.resource_config) !== undefined) {
        return config;
      }

      return {};

      
      // Resource
      /*if (isChild) return { isChild: "rover-for-each-child-resource-true" };
      if (model.resources[resourceID] && model.resources[resourceID].config) {
        let trc = {};
        if (model.resources[resourceID].config.for_each_expression) {
          trc.for_each = model.resources[resourceID].config.for_each_expression;
        }
        if (model.resources[resourceID].config.count_expression) {
          trc.count = model.resources[resourceID].config.count_expression;
        }
        return Object.assign(
          trc,
          model.resources[resourceID].config.expressions
        );
      }

      // Defaults to returning empty object
      return {};*/
    },
    getResourceChange(resourceID, model) {
      // console.log(`resourceID: ${resourceID}`);
      // console.log(model);

      let rc = {};
      if (resourceID.includes("var.")) {
        return (rc = {});
      }
      if (resourceID.includes("output.")) {
        //let id = resourceID;
        if (model.states[resourceID] && model.states[resourceID].change) {
          const c = model.states[resourceID].change;

          if (c.actions) {
            rc.action = c.actions.length > 1 ? "replace" : c.actions[0];
          }
          rc.before = c.before ? c.before : null;
          rc.after = c.after ? c.after : {};

          if (typeof rc.before === "string") {
            rc.before = {
              value: rc.before,
            };
          }

          if (typeof rc.after === "string") {
            rc.after = {
              value: rc.after,
            };
          }

          if (c["after_unknown"]) {
            rc.after["value"] = { unknown: true };
          }

          // console.log(rc);

          return rc;
        }
        return (rc = {});
      }

      // Resource
      if (model.states[resourceID] && model.states[resourceID].change) {
        const c = model.states[resourceID].change;

        if (c.actions) {
          rc.action = c.actions.length > 1 ? "replace" : c.actions[0];
        }
        rc.before = c.before ? c.before : {};
        rc.after = c.after ? c.after : {};

        if (c["after_unknown"]) {
          Object.keys(c["after_unknown"]).forEach(function (k) {
            rc.after[k] = { unknown: true };
          });
        }
      }

      return rc;
    },
  },
  computed: {
    resource() {
      let resource = "";

      // If no config version...
      if (this.resourceID.startsWith("Resources/")) {
        resource = this.resourceID.split("/").join(".");
      } else {
        resource = this.resourceID.split("/").slice(-2).join(".");
      }

      const rArray = resource.split(".");
      const lastIndex = rArray.length - 1;

      let resourceID = rArray.join(".");

      // If no config version..
      if (this.resourceID.startsWith("Resources/")) {
        resourceID = rArray.slice(1).join(".");
      }

      /*if (
        rArray[lastIndex - 1] == "output" &&
        !resourceID.startsWith("output.")
      ) {
        resourceID = `output.${resourceID}`;
      }

      if (rArray[lastIndex - 1] == "local") {
        resourceID = `local.${rArray[lastIndex]}`;
      }

      if (rArray[lastIndex - 1] == "var") {
        resourceID = `var.${rArray[lastIndex]}`;
      }

      // If resourceID is a child only (no . in id)
      if (resourceID.match(/^[\w-]+[[]/g) != null) {
        resourceID = rArray.slice(1).join(".");
      }*/

      return {
        fileName: `${rArray[0]}.${rArray[1]}`,
        id: resourceID,
        resource_type: rArray[lastIndex - 1],
        resource_name: rArray[lastIndex],
      };
    },
    primitiveType() {
      switch (this.resource.resource_type) {
        case "output":
        case "var":
        case "local":
          return this.resource.resource_type;
        default:
          if (this.resource.id.startsWith("data.")) {
            return "data";
          }
          return "resource";
      }
    },
    isChild() {
      return this.resource.id.match(/\[[^[\]]*\]$/g) != null;
    },
    hasNoState() {
      return this.resource.id.includes("var.");
    },
    resourceConfig() {


      return this.getResourceConfig(this.resource.id, this.overview, this.isChild)

      /*if (this.resource.id === "") {
        return { action: "", before: {} };
      }

      if (!this.isChild) {
        // If it's part of a module
        if (this.resource.id.startsWith("module.")) {
          return this.getResourceConfig(
            this.resource.id,
            this.overview.resources[this.resource.parentID].module_config,
            false
          );
        }
        return this.getResourceConfig(this.resource.id, this.overview, false);
      }

      // If it's part of a module
      if (this.resource.id.startsWith("module.")) {
        return this.getResourceConfig(
          this.resource.id,
          this.overview.resources[this.resource.parentID].module_config,
          true
        );
      }
      return this.getResourceConfig(this.resource.id, this.overview, false);
      // return this.isChild;*/
    },
    resourceChange() {

      return this.getResourceChange(this.resource.id, this.overview);

    },
  },
  watch: {
    resourceID: function (newVal) {
      if (newVal.includes("var.")) {
        this.curTab = "config";
      }
    },
  },
  mounted() {
    // if rso.js file is present (standalone mode)
    // eslint-disable-next-line no-undef
    if (typeof rso !== "undefined") {
      // eslint-disable-next-line no-undef
      this.overview = rso;
    } else {
      axios.get(`/api/rso`).then((response) => {
        this.overview = response.data;
        //console.log(this.overview);
      });
    }
  },
};
</script>

<style scoped>
#resource-details {
  position: sticky;
  top: 1em;
  min-width: 0;
  /* background-color: #292a34; */
}
.tab-container {
  max-height: 70vh;
  overflow: scroll;
}
fieldset {
  margin-bottom: 2em;
}
.tabs a:hover {
  cursor: pointer;
}

.dark .tabs a {
  color:#f4ecff
}

.resource-detail {
  padding: 1em 0;
}
.tab-container {
  padding: 1em 0;
}
.tabs .disabled:hover {
  cursor: not-allowed;
  border-bottom: 4px solid var(--color-lightGrey);
}

p {
  word-break: break-all;
  white-space: normal;
}

a {
  font-weight: bold;
  border-width: 4px !important;
}

.key {
  font-weight: bold;
  font-size: 0.9em;
  text-transform: uppercase;
  margin: 0;
}

dd {
  display: inline-block;
}

dt.value {
  margin: 0.5em 0 1em 0;
  padding: 0.5em;
  font-size: 1em;
  background-color: #f4ecff;
  color: black;
  display: flex;
  align-items: center;
  justify-content: space-between;
  white-space: pre-wrap;
  overflow-x: overlay;
}

dt.value-before {
  margin: 0.5em 0 1em 0;
  padding: 0.5em;
  font-size: 1em;
  background-color: #f4ecff;
  color: rgb(166, 1, 1);
  display: flex;
  align-items: center;
  justify-content: space-between;
  white-space: pre-wrap;
  overflow-x: overlay;
}

dt.value-after {
  margin: 0.5em 0 1em 0;
  padding: 0.5em;
  font-size: 1em;
  background-color: #f4ecff;
  color: rgb(33, 142, 0);
  display: flex;
  align-items: center;
  justify-content: space-between;
  white-space: pre-wrap;
  overflow-x: overlay;
}

.resource-id {
  word-wrap: break-word;
  overflow: hidden;
  width: 100%;
}

.resource-action {
  float: right;
}

.is-child-resource {
  display: block;
  text-align: center;
  font-weight: bold;
  font-style: italic;
}

.unknown-value {
  text-align: center;
  font-weight: bold;
  font-style: italic;
}

.copy-button {
  font-size: 0.9em;
  padding: 1rem;
  align-items: flex-end;
  background-color: #8450ba;
  color: white;
  font-weight: bold;
  white-space: normal;
}

.copy-button:hover {
  cursor: pointer;
}
</style>