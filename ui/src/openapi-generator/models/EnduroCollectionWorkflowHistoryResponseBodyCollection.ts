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
import {
    EnduroCollectionWorkflowHistoryResponseBody,
    EnduroCollectionWorkflowHistoryResponseBodyFromJSON,
    EnduroCollectionWorkflowHistoryResponseBodyFromJSONTyped,
    EnduroCollectionWorkflowHistoryResponseBodyToJSON,
} from './';

/**
 * EnduroCollection-Workflow-HistoryCollectionResponseBody is the result type for an array of EnduroCollection-Workflow-HistoryResponseBody (default view)
 * @export
 * @interface EnduroCollectionWorkflowHistoryResponseBodyCollection
 */
export interface EnduroCollectionWorkflowHistoryResponseBodyCollection extends Array<EnduroCollectionWorkflowHistoryResponseBody> {
}

export function EnduroCollectionWorkflowHistoryResponseBodyCollectionFromJSON(json: any): EnduroCollectionWorkflowHistoryResponseBodyCollection {
    return EnduroCollectionWorkflowHistoryResponseBodyCollectionFromJSONTyped(json, false);
}

export function EnduroCollectionWorkflowHistoryResponseBodyCollectionFromJSONTyped(json: any, ignoreDiscriminator: boolean): EnduroCollectionWorkflowHistoryResponseBodyCollection {
    return json;
}

export function EnduroCollectionWorkflowHistoryResponseBodyCollectionToJSON(value?: EnduroCollectionWorkflowHistoryResponseBodyCollection | null): any {
    return value;
}


