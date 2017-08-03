// Code generated by go-swagger; DO NOT EDIT.

package rest

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"encoding/json"
)

// SwaggerJSON embedded version of the swagger document used at generation time
var SwaggerJSON json.RawMessage

func init() {
	SwaggerJSON = json.RawMessage([]byte(`{
  "consumes": [
    "application/json"
  ],
  "produces": [
    "application/json"
  ],
  "swagger": "2.0",
  "info": {
    "title": "Kubernikus",
    "version": "1.0.0"
  },
  "paths": {
    "/api/": {
      "get": {
        "summary": "List available api versions",
        "operationId": "ListAPIVersions",
        "responses": {
          "200": {
            "description": "OK",
            "schema": {
              "$ref": "#/definitions/ApiVersions"
            }
          },
          "401": {
            "description": "Unauthorized"
          }
        }
      }
    },
    "/api/v1/clusters/": {
      "get": {
        "summary": "List available clusters",
        "operationId": "ListClusters",
        "security": [
          {
            "keystone": []
          }
        ],
        "responses": {
          "200": {
            "description": "OK",
            "schema": {
              "type": "array",
              "items": {
                "$ref": "#/definitions/Cluster"
              }
            }
          },
          "default": {
            "$ref": "#/responses/errorResponse"
          }
        }
      },
      "post": {
        "summary": "Create a cluster",
        "operationId": "CreateCluster",
        "security": [
          {
            "keystone": []
          }
        ],
        "parameters": [
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {
              "$ref": "#/definitions/Cluster"
            }
          }
        ],
        "responses": {
          "200": {
            "description": "OK",
            "schema": {
              "$ref": "#/definitions/Cluster"
            }
          },
          "default": {
            "$ref": "#/responses/errorResponse"
          }
        }
      }
    },
    "/api/v1/clusters/{name}": {
      "get": {
        "summary": "Show the specified cluser",
        "operationId": "ShowCluster",
        "security": [
          {
            "keystone": []
          }
        ],
        "responses": {
          "200": {
            "description": "OK",
            "schema": {
              "$ref": "#/definitions/Cluster"
            }
          },
          "default": {
            "$ref": "#/responses/errorResponse"
          }
        }
      },
      "delete": {
        "summary": "Terminate the specified cluser",
        "operationId": "TerminateCluster",
        "security": [
          {
            "keystone": []
          }
        ],
        "responses": {
          "200": {
            "description": "OK",
            "schema": {
              "$ref": "#/definitions/Cluster"
            }
          },
          "default": {
            "$ref": "#/responses/errorResponse"
          }
        }
      },
      "patch": {
        "summary": "Patch the specified cluser",
        "operationId": "PatchCluster",
        "security": [
          {
            "keystone": []
          }
        ],
        "parameters": [
          {
            "uniqueItems": true,
            "type": "string",
            "name": "name",
            "in": "path",
            "required": true
          },
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {
              "$ref": "#/definitions/Cluster"
            }
          }
        ],
        "responses": {
          "200": {
            "description": "OK",
            "schema": {
              "$ref": "#/definitions/Cluster"
            }
          },
          "default": {
            "$ref": "#/responses/errorResponse"
          }
        }
      },
      "parameters": [
        {
          "uniqueItems": true,
          "type": "string",
          "name": "name",
          "in": "path",
          "required": true
        }
      ]
    }
  },
  "definitions": {
    "ApiVersions": {
      "required": [
        "versions"
      ],
      "properties": {
        "versions": {
          "description": "versions are the api versions that are available.",
          "type": "array",
          "items": {
            "type": "string"
          }
        }
      }
    },
    "Cluster": {
      "type": "object",
      "properties": {
        "name": {
          "description": "name of the cluster",
          "type": "string",
          "pattern": "^[a-z]([-a-z0-9]*[a-z0-9])?$"
        },
        "status": {
          "description": "status of the cluster",
          "type": "string"
        }
      }
    },
    "Principal": {
      "type": "object",
      "properties": {
        "account": {
          "description": "account id",
          "type": "string"
        },
        "id": {
          "description": "userid",
          "type": "string"
        },
        "name": {
          "description": "username",
          "type": "string"
        },
        "roles": {
          "description": "list of roles the user has in the given scope",
          "type": "array",
          "items": {
            "type": "string"
          }
        }
      }
    },
    "error": {
      "description": "the error model is a model for all the error responses coming from kvstore\n",
      "type": "object",
      "required": [
        "message",
        "code"
      ],
      "properties": {
        "cause": {
          "$ref": "#/definitions/error"
        },
        "code": {
          "description": "The error code",
          "type": "integer",
          "format": "int64"
        },
        "helpUrl": {
          "description": "link to help page explaining the error in more detail",
          "type": "string",
          "format": "uri"
        },
        "message": {
          "description": "The error message",
          "type": "string"
        }
      }
    }
  },
  "responses": {
    "errorResponse": {
      "description": "Error",
      "schema": {
        "$ref": "#/definitions/error"
      }
    }
  },
  "securityDefinitions": {
    "keystone": {
      "description": "OpenStack Keystone Authentication",
      "type": "apiKey",
      "name": "x-auth-token",
      "in": "header"
    }
  }
}`))
}
