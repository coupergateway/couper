---
title: 'Merge of Configuration File(s)'
description: 'For more complex situations you can configure Couper with multiple files.'
---

# Merging

Couper supports merging of blocks and attributes from multiple configuration files
(see [Basic Options of Command Line Interface](/configuration/command-line#basic-options)).

* [Merging](#merging)
* [General Rules of Merging](#general-rules-of-merging)
* [Merging of `server` Blocks](#merging-of-server-blocks)
* [Merging of `definitions` Blocks](#merging-of-definitions-blocks)
* [Merging of `defaults` Blocks](#merging-of-defaults-blocks)
* [Merging of `settings` Blocks](#merging-of-settings-blocks)

## General Rules of Merging

When merging, all attributes (except `environment_variables` in the `defaults` block) replace existing attributes with the same name, if any, otherwise they are added.

Blocks that cannot have labels (eg. `cors`, `files` etc.) replace existing blocks with the same name, if any, otherwise they are added.

Blocks with optional labels (eg. `server`, `api`, `spa`, `files` etc.) are merged recursively with blocks with the same label (blocks without a label are merged with blocks with the same name and no label in each context), if any, otherwise they are added. Only one unlabeled block of the same type is allowed in each context (eg. `api` blocks in a `server` block).

Blocks with required label (eg. `endpoint`) replace existing blocks with the same name and label in each context, if any, otherwise they are added.

Blocks with (optional) multiple labels (eg. `error_handler`) replace existing blocks with identical labels, if any, otherwise they are added.

Currently, here is no way to remove an attribute or a block from the configuration.

## Merging of `server` Blocks

* When `server` blocks are merged:
  * All attributes replace existing attributes with the same name, if any, otherwise they are added.
  * The `cors` blocks replace existing blocks with the same name, if any, otherwise, they are added.
  * All `endpoint` blocks replace existing blocks with the same label, if any, otherwise they are added.
* When `spa` or `files` blocks are merged:
  * All attributes replace existing attributes with the same name, if any, otherwise they are added.
  * The `cors` block replaces existing `cors` block, if any, otherwise a new `cors` block is added.
* When `api` blocks are merged:
  * All attributes replace existing attributes with the same name, if any, otherwise they are added.
  * The `cors` block replaces existing `cors` block, if any, otherwise a new `cors` block is added.
  * All `endpoint` blocks replace existing blocks with the same label, if any, otherwise they are added.
  * All `error_handler` blocks replace existing blocks with identical labels, if any, otherwise they are added.

**Note:** An `error_handler` block cannot be replaced in or added to an `endpoint` block. Therefore, the `endpoint` block must be completely replaced.

## Merging of `definitions` Blocks

* The `definitions` blocks are merged recursively, if any, otherwise a new `definitions` block is added.
* All blocks inside a `definitions` block replace existing blocks with the same name and label.

## Merging of `defaults` Blocks

* The `defaults` blocks are merged recursively, if any, otherwise a new `defaults` block is added.
* All attributes inside a `defaults` block (except `environment_variables`) replace existing attributes with the same name, if any, otherwise they are added.
* Single values of an `environment_variables` attribute replace existing values with the same key, if any, otherwise they are added.

## Merging of `settings` Blocks

* The `settings` blocks are merged recursively, if any, otherwise a new `settings` block is added.
* All attributes inside a `settings` block replace existing attributes with the same name, if any, otherwise they are added.
