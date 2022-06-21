// File: contracts/library/Ownable.sol

pragma solidity ^0.4.24;

contract Ownable {

    address public owner;

    modifier onlyOwner {
        require(isOwner(msg.sender));
        _;
    }

    function Ownable() public {
        owner = msg.sender;
    }

    function transferOwnership(address _newOwner) public onlyOwner {
        owner = _newOwner;
    }

    function isOwner(address _address) public view returns (bool) {
        return owner == _address;
    }
}

// File: contracts/library/Pausable.sol

pragma solidity ^0.4.24;


/**
 * @title Pausable
 * @dev Base contract which allows children to implement an emergency stop mechanism.
 */
contract Pausable is Ownable {
  event Pause();
  event Unpause();

  bool public paused = false;


  /**
   * @dev Modifier to make a function callable only when the contract is not paused.
   */
  modifier whenNotPaused() {
    require(!paused);
    _;
  }

  /**
   * @dev Modifier to make a function callable only when the contract is paused.
   */
  modifier whenPaused() {
    require(paused);
    _;
  }

  /**
   * @dev called by the owner to pause, triggers stopped state
   */
  function pause() onlyOwner whenNotPaused public {
    paused = true;
    emit Pause();
  }

  /**
   * @dev called by the owner to unpause, returns to normal state
   */
  function unpause() onlyOwner whenPaused public {
    paused = false;
    emit Unpause();
  }
}

// File: contracts/verifiers/Verifier.sol

pragma solidity ^0.4.24;

interface Verifier {
    function description() external view returns (string);
    function verify(bytes) external returns (bool);
}

// File: contracts/DelegateProfile.sol

pragma solidity ^0.4.24;



interface Register {
    function addrToIdx(address) external view returns (uint256);
}
/**
 * @title Register delegate details.
 * @author IoTeX Team
 */
contract DelegateProfile is Pausable {
    event FeeUpdated(uint256 fee);
    event ProfileUpdated(address delegate, string name, bytes value);
    event FieldDeprecated(string name);
    event NewField(string name);

    struct Field {
        Verifier verifier;
        bool deprecated;
        bool flag;
    }

    modifier onlyRegistered() {
        require(_registered(msg.sender), "not registered");
        _;
    }

    Register public register;
    mapping(address => mapping(string => bytes)) private profiles;

    struct Profile {
        mapping(string => bytes) values;
        bool flag;
    }
    string[] public fieldNames;
    mapping(string => Field) private fields;

    constructor(address registerAddr) public {
        register = Register(registerAddr);
    }

    function _registered(address _addr) private view returns (bool) {
        if (address(register) == address(0)) {
            return true;
        }
        return register.addrToIdx(_addr) > 0;
    }

    function deprecated(string _name) internal view returns (bool) {
        require(fields[_name].flag, "undefined field name");
        return fields[_name].deprecated;
    }

    function numOfFields() public view returns (uint256) {
        return fieldNames.length;
    }

    function getFieldByIndex(uint256 _idx) public view returns (string name_, Verifier verifier_, bool deprecated_) {
        require(_idx < numOfFields(), "field index out of boundary");
        name_ = fieldNames[_idx];
        verifier_ = fields[name_].verifier;
        deprecated_ = fields[name_].deprecated;
    }

    function getFieldByName(string _name) public view returns (Verifier verifier_, bool deprecated_) {
        require(fields[_name].flag, "undefined field name");
        verifier_ = fields[_name].verifier;
        deprecated_ = fields[_name].deprecated;
    }

    function newFieldInternal(string _name, Verifier _verifier, bool _deprecated) internal {
        require(!fields[_name].flag, "duplicate field name");
        fields[_name] = Field(_verifier, _deprecated, true);
        fieldNames.push(_name);
        emit NewField(_name);
    }

    function newField(string _name, address _verifierAddr) public onlyOwner {
        require(bytes(_name).length > 0, "field name cannot be empty");
        newFieldInternal(_name, Verifier(_verifierAddr), false);
    }

    function deprecateField(string _name) public onlyOwner {
        require(fields[_name].flag, "undefined field");
        fields[_name].deprecated = true;
        emit FieldDeprecated(_name);
    }

    function withdraw() public onlyOwner {
        msg.sender.transfer(address(this).balance);
    }

    function updateProfileInternal(address _delegate, string _name, bytes _value) internal {
        require(_value.length < 65536, "value too long");
        require(!deprecated(_name), "deprecated field");
        require(fields[_name].verifier.verify(_value), "invalid value");
        profiles[_delegate][_name] = _value;
        emit ProfileUpdated(_delegate, _name, _value);
    }

    function updateProfile(string _name, bytes _value) public onlyRegistered whenNotPaused {
        updateProfileInternal(msg.sender, _name, _value);
    }

    function updateProfileForDelegate(address _delegate, string _name, bytes _value) public onlyOwner {
        require(_registered(_delegate), "not registered");
        updateProfileInternal(_delegate, _name, _value);
    }

    function updateProfileWithByteCodeInternal(address delegate, bytes memory _byteCode) internal {
        uint256 l;
        string memory name;
        bytes memory value;
        uint256 i = 0;
        while (i < _byteCode.length) {
            // load name length
            l = toUint(_byteCode, i);
            require(l > 0, "invalid length");
            i += 32;
            // load name
            name = string(slice(_byteCode, i, l));
            i += l;
            // load value length
            l = toUint(_byteCode, i);
            i += 32;
            // load value
            value = slice(_byteCode, i, l);
            i += l;
            updateProfileInternal(delegate, name, value);
        }
    }

    function updateProfileWithByteCode(bytes memory _byteCode) public onlyRegistered whenNotPaused {
        updateProfileWithByteCodeInternal(msg.sender, _byteCode);
    }

    function updateProfileWithByteCodeForDelegate(address _delegate, bytes _byteCode) public onlyOwner {
        updateProfileWithByteCodeInternal(_delegate, _byteCode);
    }

    function getProfileByField(address _delegate, string _field) public view returns (bytes) {
        require(_registered(_delegate), "not registered");
        return profiles[_delegate][_field];
    }

    function getEncodedProfile(address _delegate) public view returns (bytes code_) {
        require(_registered(_delegate), "not registered");
        uint256 l = 0;
        uint256 i;
        string memory fieldName;
        for (i = 0; i < fieldNames.length; i++) {
            fieldName = fieldNames[i];
            if (!deprecated(fieldName)) {
                l += 64 + bytes(fieldName).length + profiles[_delegate][fieldName].length;
            }
        }
        code_ = new bytes(l);
        uint256 codePtr;
        // solium-disable-next-line security/no-inline-assembly
        assembly {
            codePtr := add(code_, 32)
        }
        for (i = 0; i < fieldNames.length; i++) {
            fieldName = fieldNames[i];
            if (!deprecated(fieldName)) {
                codePtr = copy(codePtr, bytes(fieldName));
                codePtr = copy(codePtr, profiles[_delegate][fieldName]);
            }
        }
    }

    function toUint(bytes memory _bytes, uint _start) internal pure returns (uint256 retval_) {
        require(_bytes.length >= (_start + 32), "invalid byte code");
        // solium-disable-next-line security/no-inline-assembly
        assembly {
            retval_ := mload(add(add(_bytes, 32), _start))
        }
    }

    function copy(uint _dest, bytes memory _src) internal pure returns (uint256) {
        uint256 l = _src.length + 32;
        uint256 offset = 0;
        for (; offset + 32 <= l; offset += 32) {
            // solium-disable-next-line security/no-inline-assembly
            assembly {
                mstore(add(_dest, offset), mload(add(_src, offset)))
            }
        }
        uint256 mask = 256 ** (32 - l % 32) - 1;
        // solium-disable-next-line security/no-inline-assembly
        assembly {
            let srcpart := and(mload(add(_src, offset)), not(mask))
            let destpart := and(mload(add(_dest, offset)), mask)
            mstore(add(_dest, offset), or(destpart, srcpart))
        }
        return _dest + 32 + _src.length;
    }

    function slice(bytes memory _bytes, uint _start, uint _length) internal pure returns (bytes memory) {
        require(_bytes.length >= (_start + _length), "invalid byte code");
        bytes memory tempBytes;
        // solium-disable-next-line security/no-inline-assembly
        assembly {
            switch iszero(_length)
            case 0 {
                // Get a location of some free memory and store it in tempBytes as
                // Solidity does for memory variables.
                tempBytes := mload(0x40)

                // The first word of the slice result is potentially a partial
                // word read from the original array. To read it, we calculate
                // the length of that partial word and start copying that many
                // bytes into the array. The first word we copy will start with
                // data we don't care about, but the last `lengthmod` bytes will
                // land at the beginning of the contents of the new array. When
                // we're done copying, we overwrite the full first word with
                // the actual length of the slice.
                let lengthmod := and(_length, 31)

                // The multiplication in the next line is necessary
                // because when slicing multiples of 32 bytes (lengthmod == 0)
                // the following copy loop was copying the origin's length
                // and then ending prematurely not copying everything it should.
                let mc := add(add(tempBytes, lengthmod), mul(0x20, iszero(lengthmod)))
                let end := add(mc, _length)

                for {
                    // The multiplication in the next line has the same exact purpose
                    // as the one above.
                    let cc := add(add(add(_bytes, lengthmod), mul(0x20, iszero(lengthmod))), _start)
                } lt(mc, end) {
                    mc := add(mc, 0x20)
                    cc := add(cc, 0x20)
                } {
                    mstore(mc, mload(cc))
                }

                mstore(tempBytes, _length)

                //update free-memory pointer
                //allocating the array padded to 32 bytes like the compiler does now
                mstore(0x40, and(add(mc, 31), not(31)))
            }
            //if we want a zero-length slice let's just return a zero-length array
            default {
                tempBytes := mload(0x40)

                mstore(0x40, add(tempBytes, 0x20))
            }
        }

        return tempBytes;
    }
}
