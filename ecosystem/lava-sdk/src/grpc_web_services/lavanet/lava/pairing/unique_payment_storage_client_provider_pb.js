/**
 * @fileoverview
 * @enhanceable
 * @suppress {messageConventions} JS Compiler reports an error if a variable or
 *     field starts with 'MSG_' and isn't a translatable message.
 * @public
 */
// GENERATED CODE -- DO NOT EDIT!

var jspb = require('google-protobuf');
var goog = jspb;
var global = Function('return this')();

goog.exportSymbol('proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider', null, global);

/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider.displayName = 'proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider';
}


if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto suitable for use in Soy templates.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     com.google.apps.jspb.JsClassTemplate.JS_RESERVED_WORDS.
 * @param {boolean=} opt_includeInstance Whether to include the JSPB instance
 *     for transitional soy proto support: http://goto/soy-param-migration
 * @return {!Object}
 */
proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider.prototype.toObject = function(opt_includeInstance) {
  return proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Whether to include the JSPB
 *     instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider.toObject = function(includeInstance, msg) {
  var f, obj = {
    index: jspb.Message.getFieldWithDefault(msg, 1, ""),
    block: jspb.Message.getFieldWithDefault(msg, 2, 0),
    usedcu: jspb.Message.getFieldWithDefault(msg, 3, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider}
 */
proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider;
  return proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider}
 */
proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setIndex(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readUint64());
      msg.setBlock(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readUint64());
      msg.setUsedcu(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getIndex();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getBlock();
  if (f !== 0) {
    writer.writeUint64(
      2,
      f
    );
  }
  f = message.getUsedcu();
  if (f !== 0) {
    writer.writeUint64(
      3,
      f
    );
  }
};


/**
 * optional string index = 1;
 * @return {string}
 */
proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider.prototype.getIndex = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/** @param {string} value */
proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider.prototype.setIndex = function(value) {
  jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional uint64 block = 2;
 * @return {number}
 */
proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider.prototype.getBlock = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/** @param {number} value */
proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider.prototype.setBlock = function(value) {
  jspb.Message.setProto3IntField(this, 2, value);
};


/**
 * optional uint64 usedCU = 3;
 * @return {number}
 */
proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider.prototype.getUsedcu = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/** @param {number} value */
proto.lavanet.lava.pairing.UniquePaymentStorageClientProvider.prototype.setUsedcu = function(value) {
  jspb.Message.setProto3IntField(this, 3, value);
};


goog.object.extend(exports, proto.lavanet.lava.pairing);
