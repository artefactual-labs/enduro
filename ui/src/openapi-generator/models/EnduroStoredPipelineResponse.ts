/* tslint:disable */
/* eslint-disable */
/**
 * Enduro API
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * The version of the OpenAPI document: 
 * 
 *
 * NOTE: This class is auto generated by OpenAPI Generator (https://openapi-generator.tech).
 * https://openapi-generator.tech
 * Do not edit the class manually.
 */

import { exists, mapValues } from '../runtime';
/**
 * StoredPipeline describes a pipeline retrieved by this service. (default view)
 * @export
 * @interface EnduroStoredPipelineResponse
 */
export interface EnduroStoredPipelineResponse {
    /**
     * Name of the collection
     * @type {string}
     * @memberof EnduroStoredPipelineResponse
     */
    id?: string;
    /**
     * Name of the collection
     * @type {string}
     * @memberof EnduroStoredPipelineResponse
     */
    name: string;
}

export function EnduroStoredPipelineResponseFromJSON(json: any): EnduroStoredPipelineResponse {
    return EnduroStoredPipelineResponseFromJSONTyped(json, false);
}

export function EnduroStoredPipelineResponseFromJSONTyped(json: any, ignoreDiscriminator: boolean): EnduroStoredPipelineResponse {
    if ((json === undefined) || (json === null)) {
        return json;
    }
    return {
        
        'id': !exists(json, 'id') ? undefined : json['id'],
        'name': json['name'],
    };
}

export function EnduroStoredPipelineResponseToJSON(value?: EnduroStoredPipelineResponse | null): any {
    if (value === undefined) {
        return undefined;
    }
    if (value === null) {
        return null;
    }
    return {
        
        'id': value.id,
        'name': value.name,
    };
}


