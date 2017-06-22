/* Digital Rebar: Provision */
/* Copyright 2016, RackN */
/* License: Apache v2 */
/* jshint esversion: 6 */

function debounce(func, wait, immediate) {
  var timeout;
  return function() {
    var context = this, args = arguments;
    var later = function() {
      timeout = null;
      if (!immediate) func.apply(context, args);
    };
    var callNow = immediate && !timeout;
    clearTimeout(timeout);
    timeout = setTimeout(later, wait);
    if (callNow) func.apply(context, args);
  };
};

class Subnet extends React.Component {

  constructor(props) {
    super(props);

    this.toggleExpand = this.toggleExpand.bind(this);
    this.handleChange = this.handleChange.bind(this);
    this.handleOptionChange = this.handleOptionChange.bind(this);
    this.update = this.update.bind(this);
    this.remove = this.remove.bind(this);
  }

  // expands this subnet
  toggleExpand() {
    var subnet = this.props.subnet;
    subnet._expand = !subnet._expand;
    this.props.change(this.props.index, subnet);
  }

  // gets the name of an option from its code
  getCodeName(code) {
    var codes = {
      1: "Subnet Mask",
      3: "Default Gateway",
      6: "DNS Server",
      15: "Domain Name",
      28: "Broadcast",
      42: "NTP Server",
      67: "Bootfile Name"
    };
    return codes[code] || "Code " + code;
  }

  // called to make the post/put request that updates the subnet
  update() {
    this.props.update(this.props.index);
  }

  // makes the delete request to remove the subnet
  remove() {
    this.props.remove(this.props.index);
  }

  // called when an input changes
  handleChange(event) {
    var val = event.target.value;
    if(event.target.type === "number" && val && typeof val !== 'undefined')
      val = parseInt(val);
    if(event.target.type === "select-one") {
      val = val === "true";
    }
    var subnet = this.props.subnet;
    subnet[event.target.name] = val;
    subnet._edited = true;

    this.props.change(this.props.index, subnet);
  }

  // called when an option input is changed
  handleOptionChange(event) {
    var subnet = this.props.subnet;
    subnet.Options[event.target.name].Value = event.target.value;
    subnet._edited = true;

    this.props.change(this.props.index, subnet);
  }

  // renders the element
  render() {
    var subnet = JSON.parse(JSON.stringify(this.props.subnet));
    return (
      <tbody
        className={
          (subnet.updating ? 'updating-content' : '') + " " + (subnet._expand ? 'expanded' : '') + ' ' + (subnet._error ? 'error' : (subnet._new ? 'new' : (subnet._edited ? 'edited' : '')))}>
        <tr>
          <td>
            {subnet._new ? <input
              type="text"
              name="Name"
              size="8"
              placeholder="eth0"
              value={subnet.Name}
              onChange={this.handleChange}/> : subnet.Name}
          </td>
          <td>
            <input
              type="text"
              name="Subnet"
              size="15"
              placeholder="192.168.1.1/24"
              value={subnet.Subnet}
              onChange={this.handleChange}/>
          </td>
          <td>
            <select
              name="OnlyReservations"
              type="bool"
              value={subnet.OnlyReservations}
              onChange={this.handleChange}>
                <option value="true">Required</option>
                <option value="false">Optional</option>
              </select>
          </td>
          <td>
            <input
              type="number"
              name="ActiveLeaseTime"
              style={{width: "50px"}}
              placeholder="0"
              value={subnet.ActiveLeaseTime}
              onChange={this.handleChange}/>
            &nbsp;&amp;&nbsp;
            <input
              type="number"
              name="ReservedLeaseTime"
              style={{width: "50px"}}
              placeholder="7200"
              value={subnet.ReservedLeaseTime}
              onChange={this.handleChange}/>
              &nbsp;seconds
          </td>
          <td>
            <div>
              <input
                type="text"
                name="ActiveStart"
                size="15"
                placeholder="10.0.0.0"
                value={subnet.ActiveStart}
                onChange={this.handleChange}/>
              ...
            </div>
            <div>
              <input
                type="text"
                name="ActiveEnd"
                size="15"
                placeholder="10.0.0.255"
                value={subnet.ActiveEnd}
                onChange={this.handleChange}/>
            </div>
          </td>
          <td style={{border: 'thin solid black !important'}} className="icon-buttons">
            {subnet._new || subnet._edited ? 
            <button onClick={this.update} className="icon-button">
              save
              <span className="tooltip">{subnet._new ? 'Add' : 'Save'}</span>
            </button> : ''}
            <button onClick={this.remove} className="icon-button">
              delete
              <span className="tooltip">Remove</span>
            </button>
            <button onClick={this.props.copy} className="icon-button">
              content_copy
              <span className="tooltip">Copy</span>
            </button>
          </td>
        </tr>
        <tr>
          <td colSpan="7">
            {subnet._expand ? (<div>
              <h2>Other Values</h2>
              <table>
                <tbody>
                  <tr>
                    <td style={{textAlign: 'right', fontWeight: 'bold'}}>Next Server</td>
                    <td>
                      <input
                        type="text"
                        name="NextServer"
                        size="12"
                        value={subnet.NextServer}
                        onChange={this.handleChange}/>
                    </td>
                  </tr>
                </tbody>
              </table>
              <h2>Options</h2>
              <table>
                <tbody>
                  {subnet.Options.map((val, i) =>
                    <tr key={i}>
                      <td style={{textAlign: 'right', fontWeight: 'bold'}}>{this.getCodeName(val.Code)}</td>
                      <td>
                        <input
                          type="text"
                          name={i}
                          value={val.Value}
                          onChange={this.handleOptionChange}/>
                      </td>
                      <td>Option # {val.Code}</td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>): <span/>}
            {subnet._error && <div>
              <h2><span className="material-icons">error</span>{subnet._errorMessage}</h2>
            </div>}
            <div className="expand" onClick={this.toggleExpand}>
              {subnet._expand ? <span className="material-icons">expand_less</span> : <span className="material-icons">expand_more</span>}
            </div>
          </td>
        </tr>
      </tbody>
    );
  }
}

class Subnets extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      subnets: [],
      interfaces: [],
    };

    this.componentDidMount = this.componentDidMount.bind(this);
    this.update = this.update.bind(this);
    this.addSubnet = this.addSubnet.bind(this);
    this.updateSubnet = this.updateSubnet.bind(this);
    this.removeSubnet = this.removeSubnet.bind(this);
    this.changeSubnet = this.changeSubnet.bind(this);
  }
  
  // gets the subnet and interface json from the api
  getSubnets() {
    return new Promise((resolve, reject) => {
      var subnets = {};
      var interfaces = {};

      // get the interfaces from the api
      $.getJSON("../api/v3/interfaces", data => {
        for(var key in data) {
          var iface = data[key];
          interfaces[iface.Name] = iface;
        }

        // add get subnets from the api and update the state
        $.getJSON("../api/v3/subnets", data => {
          for(var key in data) {
            var subnet = data[key];
            subnets[subnet.Name] = subnet;
            // don't show interfaces if the name matches
            if (interfaces[subnet.Name] != undefined)
              delete interfaces[subnet.Name];
          }

          resolve({
            interfaces: interfaces,
            subnets: subnets
          });

        }).fail(() => {
          reject("Failed getting subnets");
        });

      }).fail(() => {
        reject("Failed getting interfaces");
      });
    });
  }

  // get the subnets and interfaces once this component mounts
  componentDidMount() {
    this.update();
  }

  update() {
    this.getSubnets().then(data => {
      this.setState({
        subnets: Object.keys(data.subnets).map(k => data.subnets[k]),
        interfaces: Object.keys(data.interfaces).map(k => data.interfaces[k])
      });
    }, err => {
    });
  }

  // called to create a new subnet
  // allows some data other than defaults to be passed in
  addSubnet(template) {

    function applyCIDR(cidr, ip) {
      var rangeMin = ip.split('.');
      var rangeMax = [];
      for(var i = 0; i < 4; i++) {
        var n = Math.min(cidr, 8);
        rangeMin[i] &= (256 - Math.pow(2, 8 - n));
        rangeMax[i] = rangeMin[i] + Math.pow(2, 8 - n) - 1;
        cidr -= n;
      }
      return [rangeMin.join('.'), rangeMax.join('.')];
    }

    var ip, range;
    if (template.IP) {
      var fq = template.IP.indexOf('/');
      ip = template.IP.substring(0,fq);
      var cidr = parseInt(template.IP.split('/')[1]);
      var range = applyCIDR(cidr, ip);
    }
    var subnet = {
      _new: true,
      Name: '',
      ActiveLeaseTime: 60,
      ReservedLeaseTime: 7200,
      OnlyReservations: false,
      ActiveStart: (ip ? range[0] : ''),
      Subnet: '',
      ActiveEnd: (ip ? range[1] : ''),
      Strategy: "MAC",
      NextServer: (ip || ''),
      Options: [
        {Code: 3, Value: (ip || '')},
        {Code: 6, Value: (ip || '')},
        {Code: 15, Value: 'example.com'},
        {Code: 67, Value: 'lpxelinux.0'}
      ]
    };

    // merge the template into our subnet if we have one
    if(typeof template !== "undefined") {
      for(var key in template) {
        if(key[0] === "_")
          continue;

        if(key === 'Options') {
          for(var i = 0; i < template.Options.length; i++) {
            var index = [3, 6, 15, 67].indexOf(template.Options[i].Code);
            if(index >= 0) {
              subnet.Options[index].Value = template.Options[i].Value;
            }
            else {
              subnet.Options.push(template.Options[i]);
            }
          }
        } else
          subnet[key] = template[key];
      }
    }

    // update the state
    this.setState({
      subnets: this.state.subnets.concat(subnet)
    });
  }

  // makes the post/put request to update the subnet
  // also updates the interface
  updateSubnet(i) {
    var subnet = this.state.subnets[i];
    subnet.updating = true;
    this.setState({subnet: this.state.subnets});

    $.ajax({
      type: (subnet._new ? "POST" : "PUT"),
      dataType: "json",
      contentType: "application/json",
      url: "/api/v3/subnets" + (subnet._new ? "" : "/" + subnet.Name),
      data: JSON.stringify(subnet)
    }).done((resp) => {
      
      // update the subnets list with our new interface
      var subnets = this.state.subnets.concat([]);

      resp.updating = false;
      resp._edited = false;
      resp._new = false;
      resp._error = false;
      resp._errorMessage = '';
      
      //  update the state
      subnets[i] = resp;

      // remove matching interfaces
      var interfaces = this.state.interfaces.concat([]);
      for (var index in interfaces) {
        if (interfaces[index].Name == resp.Name)
          interfaces.splice(index)
        }

      this.setState({
        subnets: subnets,
        interfaces: interfaces
      });

    }).fail((err) => {
      
      var subnets = this.state.subnets.concat([]);
      var subnet = subnets[i];
      subnet.updating = false;
      subnet._error = true;

      // If our error is from the backend
      if(err.responseText) {
        var response = JSON.parse(err.responseText);
        subnet._errorMessage = " (" + err.status + "): " + response.Messages.join(", ");
      } else { // maybe the backend is down
        subnet._errorMessage = err.status;
      }

      this.setState({
        subnets: subnets
      });
    });
  }

  // makes the delete request to remove the subnet or just deletes the new subnet
  removeSubnet(i) {
    var subnets = this.state.subnets.concat([]);
    var subnet = this.state.subnets[i];
    if(subnet._new) {
      subnets.splice(i, 1);
      this.setState({
        subnets: subnets
      });
      return;
    }
    subnets[i].updating = true;
    this.setState({subnets: subnets});

    $.ajax({
      type: "DELETE",
      dataType: "json",
      contentType: "application/json",
      url: "/api/v3/subnets/" + subnet.Name,
    }).done((resp) => {
            // update the subnets list with our new interface
      var subnets = this.state.subnets.concat([]);
      subnets.splice(i, 1);
      this.setState({
        subnets: subnets
      });

    }).fail((err) => {
      subnet.updating = false;
      subnet._error = true;
      // If our error is from the backend
      if(err.responseText) {      
        var response = JSON.parse(err.responseText);
        subnet._errorMessage = " (" + err.status + "): " + response.Messages.join(", ");
      } else { // maybe the backend is down
        subnet._errorMessage = err.status;
      }

      this.setState({
        subnets: subnets
      });
    });
  }

  // updates the state and changes a subnet at a specified index
  changeSubnet(i, subnet) {
    var subnets = this.state.subnets.concat([]);
    subnets[i] = subnet;
    this.setState({
      subnets: subnets
    });
  }

  render() {
    $('#subnetCount').text(this.state.subnets.length);
    return (
    <div id="subnets" style={{paddingTop: '51px'}}>
      <h2 style={{display: 'flex', justifyContent: 'space-between'}}>
        <span className="section-head">Subnets</span>
        <span>
          <a target="_blank" href="http://provision.readthedocs.io/en/latest/doc/ui.html#subnets">UI Help</a> | <a target="_blank" href="/swagger-ui/#/subnets">API Help</a>
        </span>
      </h2>

      <table className="fullwidth input-table">
        <thead>
          <tr>
            <th>Name/NIC</th>
            <th>Subnet</th>
            <th>Reservations</th>
            <th>Active &amp; Reserved Lease</th>
            <th>Range</th>
            <th></th>
          </tr>
        </thead>
        {this.state.subnets.map((val, i) =>
          <Subnet
            subnet={val}
            update={this.updateSubnet}
            change={this.changeSubnet}
            remove={this.removeSubnet}
            copy={()=>this.addSubnet(this.state.subnets[i])}
            key={i}
            index={i} />
        )}
        <tfoot>
          <tr>
            <td colSpan="7" style={{textAlign: "center"}}>
              <button onClick={()=>this.addSubnet({})}>New Subnet</button>
            </td>
          </tr>
        </tfoot>
      </table>

      <h2 style={{display: "flex", flexDirection: "row", alignItems: "center"}}>
        <span>Available Interfaces: </span>
        <span className="interface-list">
          {this.state.interfaces.map((val) => 
            val.Addresses.map((subval, i) =>
              <a key={val.Name+"-"+i} className="interface-pair" onClick={()=>this.addSubnet({Name: val.Name, Subnet: subval, Interface: val.Name, IP: subval})}>
                <header>{val.Name}</header>
                <subhead>{subval}</subhead>
              </a>
            )
          )}
        </span>
      </h2>
    </div>
    );
  }
}


class Token extends React.Component {

  constructor(props) {
    super(props);
    this.xhr = undefined;
    this.STATES = {
      STANDBY: 0,
      WAITING: 1,
      AUTHORIZED: 2,
      DONE: 3,
      ERROR: 4,
    };

    this.icons = [
      'mood',
      'hourglass_empty',
      'security',
      'done_all',
      'close'
    ];

    this.messages = [
      '',
      'Signing In',
      'Getting Token',
      'Done!',
      'Error'
    ];

    this.state = {
      token: '',
      username: '',
      password: '',
      useToken: false,
      code: 1,
      requestState: this.STATES.STANDBY,
    };

    this.handleChange = this.handleChange.bind(this);
    this.setToken = this.setToken.bind(this);
  }
  
  // gets the name of an option from its code
  getCodeName() {
    var codes = {
      0: "API Unreachable",
      1: " ",
      200: "Granted",
      401: "No Credentials/Token",
      403: "Invalid Credentials/Token"
    };
    return codes[this.state.code] || "Code " + this.state.code;
  }

  // update the token once this component mounts
  componentDidMount() {
    if (location.search.startsWith("?token=")) {
      var t = location.search.substring(7);
      this.setState({token: t, useToken: true});
      this.setToken(t);
    } else if (localStorage.DrAuthToken) {
      this.setState({useToken: true});
      this.setToken(localStorage.DrAuthToken);
    }
  }

  // tests a token for authenticity
  setToken(token) {
    var bootenvs = []
    var send_token = 'Bearer ' + token;
    let Token = this;

    if (token.includes(':')) // tokens are in base64 otherwise and will not include colons
      send_token = 'Basic ' + btoa(token);
    else
      localStorage.DrAuthToken = token;
    console.log(token);

    if(typeof this.xhr !== 'undefined') {
      this.xhr.abort();
      this.xhr = undefined;
    }

    $.ajaxSetup({
      headers: {
        Authorization: send_token
      }
    });

    this.xhr = $.ajax({
      url: '../api/v3/bootenvs',
      type: 'GET',
      async: true
    }).then((data) => {
      for(var key in data) {
        if (data[key].Available)
          bootenvs.push(data[key].Name)
      }


      if(token.includes(":")) {
        Token.setState({requestState: Token.STATES.AUTHORIZED});

        var name = token.split(":")[0];
        Token.xhr = $.ajax({
          url: "../api/v3/users/" + name + "/token?ttl=" + (8 * 60 * 60), // 8 hours in seconds
          type: "GET",
          dataType: "json",
          async: true,
          success(data) {

            localStorage.DrAuthToken = data.Token;
            $.ajaxSetup({
              headers: {
                Authorization: "Bearer " + data.Token
              }
            });
            Token.setState({code: 200, requestState: Token.STATES.DONE});
            Token.xhr = undefined;
            Token.props.onAccessChange(true, bootenvs);
          },
          error(xhr, status, error) {
            Token.xhr = undefined;
            Token.setState({code: xhr.status, requestState: Token.STATES.ERROR});
            localStorage.DrAuthToken = "";
            Token.props.onAccessChange(false, []);
          }
        });
      } else {
        Token.xhr = undefined;
        Token.setState({code: 200, requestState: Token.STATES.DONE});
        Token.props.onAccessChange(true, bootenvs);
      }
    }, (xhr, status, error) => {
      localStorage.DrAuthToken = "";
      Token.xhr = undefined;
      console.log(error);
      Token.setState({code: xhr.status, requestState: Token.STATES.ERROR});
      Token.props.onAccessChange(false, bootenvs);
    });
  }

  // called when an input changes
  handleChange(event) {
    let state = this.state;
    state.requestState = this.STATES.WAITING
    state[event.target.name] = event.target.value;
    this.setState(state);
    let token = this.state.useToken ? this.state.token : this.state.username + ':' + this.state.password;
    clearTimeout(this.tokenTimeout);
    let setToken = this.setToken;
    this.tokenTimeout = setTimeout(()=>{setToken(token)}, 500);
  }

  render() {
    let Token = this;
    return (
      <div>
        <div className="login-box">
          <div className="login-tabs">
            <div className={'tab ' + (this.state.useToken ? '' : 'active')} onClick={()=>this.setState({useToken: false})}>
              <i className="material-icons">person</i>
              Login
            </div>
            <div className={'tab ' + (this.state.useToken ? 'active' : '')} onClick={()=>this.setState({useToken: true})}>
              <i className="material-icons">vpn_key</i>
              Token
            </div>
          </div>
          {this.state.useToken ? (
            <div className="login-inputs">
              <div className="login-input">
                <input
                  type="text"
                  name="token"
                  size="15"
                  placeholder="Token"
                  value={this.state.token}
                  onChange={this.handleChange} />
                <i className="material-icons">vpn_key</i>
              </div>
              <div className="login-hint">
                <span>Tokens are generated via the </span>
                <a target="_blank" href="http://provision.readthedocs.io/en/stable/doc/cli/drpcli_users_token.html">drpcli binary</a>
                <span> or API</span>
              </div>
            </div>
          ) : (
            <div className="login-inputs">
              <div className="login-input">
                <input
                  type="text"
                  name="username"
                  size="15"
                  placeholder="Username"
                  value={this.state.username}
                  onChange={this.handleChange} />
                <i className="material-icons">person</i>
              </div>
              <div className="login-input">
                <input
                  type="password"
                  name="password"
                  size="15"
                  placeholder="Password"
                  value={this.state.password}
                  onChange={this.handleChange} />
                <i className="material-icons">lock</i>
              </div>
              <div className="login-hint">
                <span>Default credentials are </span>
                <code style={{textDecoration: 'underline', color: '#547f00', cursor: 'pointer'}}
                  onClick={()=>{
                    Token.setState({username:'rocketskates',password:'r0cketsk8ts'});
                    Token.setToken('rocketskates:r0cketsk8ts')
                  }}>
                  rocketskates:r0cketsk8ts
                </code>
              </div>
            </div>
          )}
          <div style={{padding: '10px', display: 'flex', alignItems: 'center', justifyContent: 'center'}}>
            <i className="material-icons">{this.icons[this.state.requestState]}</i>
            {(this.state.requestState == this.STATES.ERROR ?
              (<span style={{color: "#a00"}}>{this.getCodeName()}</span>) : 
              (<span>{this.messages[this.state.requestState]}</span>)
            )}
          </div>
        </div>
        <div className="welcome-box">
          <h1>Welcome to Digital Rebar: Provision</h1>
          <p className="description">
            <strong>DR Provision </strong> is a APLv2 simple Golang executable that provides a simple yet complete API-driven DHCP/PXE/TFTP provisioning system. It is designed to stand alone or operate as part of the <a href="http://rebar.digital/" target="_blank">Digital Rebar</a> management system. Check out some of the links below for more information!
          </p>
          <div className="welcome-menu">
            {[{
              name: 'Resources',
              links: [{
                name: 'Introduction',
                href: '',
                icon: 'book'
              }, {
                name: 'Documentation',
                href: 'http://provision.readthedocs.io/en/stable/',
                icon: 'info'
              }, {
                name: 'Videos',
                href: 'https://www.youtube.com/playlist?list=PLXPBeIrpXjfilUi7Qj1Sl0UhjxNRSC7nx',
                icon: 'video_library'
              }]
            }, {
              name: 'Support',
              links: [{
                name: 'Gitter',
                href: 'https://gitter.im/digitalrebar/core',
                icon: 'forum'
              }, {
                name: 'IRC (Freenode)',
                href: 'https://webchat.freenode.net/?channels=%23digitalrebar&uio=d4',
                icon: 'chat'
              }, {
                name: 'Mailing List',
                href: 'https://groups.google.com/forum/#!forum/digitalrebar',
                icon: 'mail'
              }]
            }, {
              name: 'Project',
              links: [{
                name: 'Contribute',
                href: 'https://github.com/digitalrebar/provision',
                icon: 'code'
              }, {
                name: 'Issues Tracker',
                href: 'https://github.com/digitalrebar/provision/issues',
                icon: 'directions'
              }]
            }].map(section =>
              <section>
                <header>{section.name}</header>
                <article>
                  {section.links.map(link =>
                    <a href={link.href} target="_blank" class="welcome-link">
                      <i className="material-icons">{link.icon}</i>
                      <span>{link.name}</span>
                    </a>
                  )}
                </article>
              </section>
            )}
          </div>
        </div>
      </div>
    );
  }
}

class Machine extends React.Component {

  constructor(props) {
    super(props);

    this.toggleExpand = this.toggleExpand.bind(this);
    this.handleChange = this.handleChange.bind(this);
    this.update = this.update.bind(this);
    this.remove = this.remove.bind(this);
  }

  // expands this subnet
  toggleExpand() {
    var machine = this.props.machine;
    machine._expand = !machine._expand;
    this.props.change(this.props.index, machine);
  }

  // called to make the post/put request that updates the subnet
  update() {
    this.props.update(this.props.index);
  }

  // makes the delete request to remove the subnet
  remove() {
    this.props.remove(this.props.index);
  }

  // called when an input changes
  handleChange(event) {
    var val = event.target.value;
    var machine = this.props.machine;
    machine[event.target.name] = val;
    machine._edited = true;

    this.props.change(this.props.index, machine);
  }

  // renders the element
  render() {
    var machine = JSON.parse(JSON.stringify(this.props.machine));
    return (
      <tbody 
        className={(machine.updating ? 'updating-content' : '') + " " + (machine._expand ? 'expanded' : '') + ' ' + (machine._error ? 'error' : (machine._new ? 'new' : (machine._edited ? 'edited' : '')))}>
        <tr>
          <td>
            <input
              type="text"
              name="Name"
              size="15"
              placeholder="name.example.com"
              value={machine.Name}
              onChange={this.handleChange} />
          </td>
          <td>
            <input
              type="text"
              name="Address"
              size="15"
              placeholder="0.0.0.0"
              value={machine.Address}
              onChange={this.handleChange} />
          </td>
          <td>
            <select
              name="BootEnv"
              type="bool"
              value={machine.BootEnv}
              onChange={this.handleChange}>
                { this.props.bootenvs.map((val) =>
                  <option key={val} value={val}>{val}</option>
                )}
            </select>
          </td>
          <td>
            <input
              type="text"
              name="Description"
              size="15"
              placeholder=""
              value={machine.Description}
              onChange={this.handleChange} />            
          </td>
          <td>
            {( machine.Uuid ?
              <div title={machine.Uuid}>
                {machine.Uuid.slice(0,4)}...{machine.Uuid.slice(-4)}
              </div>
              : "not set" )}
          </td>
          <td className="icon-buttons">
            {machine._new || machine._edited ? 
            <button onClick={this.update} className="icon-button">
              save
              <span className="tooltip">{machine._new ? 'Add' : 'Save'}</span>
            </button> : ''}
            <button onClick={this.remove} className="icon-button">
              delete
              <span className="tooltip">Remove</span>
            </button>
          </td>
        </tr>
        <tr>
          <td colSpan="6">
            {machine._expand ? (<div>
              {machine._error && <div>
                <h2><span className="material-icons">error</span>{machine._errorMessage}</h2>
              </div>}
              <h2>Template Errors</h2>
              {(machine.Errors ? machines.Errors : "none.")}
              <h2>Template Params</h2>
              {(machine.Params ? machines.Params : "none.")}
            </div>): <span/>}
            <div className="expand" onClick={this.toggleExpand}>
              {machine._expand ? <span className="material-icons">expand_less</span> : <span className="material-icons">expand_more</span>}
            </div>
          </td>
        </tr>
      </tbody>
    );
  }
}

class Machines extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      machines: [],
    };

    this.componentDidMount = this.componentDidMount.bind(this);
    this.update = this.update.bind(this);
    this.addMachine = this.addMachine.bind(this);
    this.changeMachine = this.changeMachine.bind(this);
    this.removeMachine = this.removeMachine.bind(this);
    this.updateMachine = this.updateMachine.bind(this);
  }
  
  // gets the machine json from the api
  getMachines() {
    return new Promise((resolve, reject) => {
      var bootenvs = [];

      // get the interfaces from the api
      $.getJSON("../api/v3/machines", data => {

        resolve({
          machines: data
        });

      }).fail(() => {
        reject("Failed getting Machines");
      });

    });
  }

  // get the machine once this component mounts
  componentDidMount() {
    this.update();
  }

  update() {
    this.getMachines().then(data => {
      this.setState({
        machines: data.machines
      });
    }, err => { 
    });
  }

 // called to create a new machine
 // allows some data other than defaults to be passed in
  addMachine() {
    var machine = {
      Name: "",
      Address: "0.0.0.0",
      BootEnv: "ignore",
      Description: "",
      Uuid: null,
      _new: true
    };
    // update the state
    this.setState({
      machines: this.state.machines.concat(machine)
    });
  }

  // makes the post/put request to update the machine
  updateMachine(i) {
    var machine = this.state.machines[i];
    machine.updating = true;
    this.setState({machine: this.state.machines});

    $.ajax({
      type: (machine._new ? "POST" : "PUT"),
      dataType: "json",
      contentType: "application/json",
      url: "/api/v3/machines" + (machine._new ? "" : "/" + machine.Uuid),
      data: JSON.stringify(machine)
    }).done((resp) => {
      
      // update the machines list with our new machine
      var machines = this.state.machines.concat([]);
      resp.updating = false;
      resp._edited = false;
      resp._new = false;
      resp._error = false;
      resp._errorMessage = '';
      
      //  update the state
      machines[i] = resp;
      this.setState({
        machines: machines
      });

    }).fail((err) => {
      
      var machines = this.state.machines.concat([]);
      var machine = machines[i];
      machine.updating = false;
      machine._error = true;
      machine._expand = true;

      // If our error is from the backend
      if(err.responseText) {
        var response = JSON.parse(err.responseText);
        machine._errorMessage = " (" + err.status + "): " + response.Messages.join(", ");
      } else { // maybe the backend is down
        machine._errorMessage = err.status;
      }

      this.setState({
        machines: machines
      });
    });
  }

  removeMachine(i) {
    var machines = this.state.machines.concat([]);
    var machine = this.state.machines[i];
    if (machine._new) {
      machines.splice(i,1);
      this.setState({machines: machines});
      return;
    }
    machines[i].updating = true;

    $.ajax({
      type: "DELETE",
      dataType: "json",
      contentType: "application/json",
      url: "/api/v3/machines/" + machine.Uuid,
    }).done((resp) => {
      // update the subnets list with our new interface
      var machines = this.state.machines.concat([]);
      machines.splice(i, 1);
      this.setState({
        machines: machines
      });

    }).fail((err) => {
      machine.updating = false;
      machine._error = true;
      // If our error is from the backend
      if(err.responseText) {
        var response = JSON.parse(err.responseText);
        machine._errorMessage = " (" + err.status + "): " + response.Messages.join(", ");
      } else { // maybe the backend is down
        machine._errorMessage = err.status;
      }

      this.setState({
        machines: machines
      });
    });
  }

  // updates the state and changes a machine at a specified index
  changeMachine(i, machine) {
    var machines = this.state.machines.concat([]);
    machines[i] = machine;
    this.setState({
      machines: machines
    });
  }

  render() {
    $('#machineCount').text(this.state.machines.length);
    return (
    <div id="machines" style={{paddingTop: '51px'}}>
      <h2 style={{display: 'flex', justifyContent: 'space-between'}}>
        <span className="section-head">Machines</span>
        <span>
          <a target="_blank" href="http://provision.readthedocs.io/en/latest/doc/ui.html#machines">UI Help</a> | <a target="_blank" href="/swagger-ui/#/machines">API Help</a>
        </span>
      </h2>
      <table className="fullwidth input-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Address</th>
            <th>BootEnv</th>
            <th>Description</th>
            <th>Uuid</th>
            <th></th>
          </tr>
        </thead>
        {this.state.machines.map((val, i) =>
          <Machine
            machine={val}
            bootenvs={this.props.bootenvs}
            update={this.updateMachine}
            change={this.changeMachine}
            remove={this.removeMachine}
            key={i}
            index={i}
          />
        )}
        <tfoot>
          <tr>
            <td colSpan="6" style={{textAlign: "center"}}>
              <button onClick={()=>this.addMachine({})}>New Machine</button>
            </td>
          </tr>
        </tfoot>
      </table>
    </div>
    );
  }
}

class Prefs extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      prefs: {},
      updated: false
    };

    this.componentDidMount = this.componentDidMount.bind(this);
    this.update = this.update.bind(this);
    this.changePrefs = this.changePrefs.bind(this);
    this.updatePrefs = this.updatePrefs.bind(this);
    this.handleChange = this.handleChange.bind(this);
  }

  // gets the machine json from the api
  getPrefs() {
    return new Promise((resolve, reject) => {
      $.getJSON("../api/v3/prefs", data => {
        resolve({
          prefs: data
        });
      }).fail(() => {
        reject("Failed getting Prefs");
      });
    });
  }

  // get the machine once this component mounts
  componentDidMount() {
    this.update();
  }

  update() {
    this.getPrefs().then(data => {
      this.setState({
        prefs: data.prefs
      });
    }, err => {
    });
  }

  // makes the put request to update the param
  updatePrefs() {
    var prefs = this.state.prefs;

    $.ajax({
      type: "POST",
      dataType: "json",
      contentType: "application/json",
      url: "/api/v3/prefs",
      data: JSON.stringify(prefs)
    }).done((resp) => {
      this.setState({
        prefs: prefs,
        updated: false
      });
    }).fail((err) => {
      this.onAccessChange(false);
    });
  }

  // updates the state and changes a param at a specified index
  changePrefs(name, value) {
    var prefs = this.state.prefs;
    prefs[name] = value;
    this.setState({
      prefs: prefs,
      updated: true
    });
  }

  // called when an option input is changed
  handleChange(event) {
    var name = event.target.name;
    var val = event.target.value;
    this.changePrefs(name, val);
  }

  render() {
    return (
    <div>
      <h2 style={{display: 'flex', justifyContent: 'space-between'}}>
      <span className="section-head">Preferences</span>
      <span>
        <a target="_blank" href="http://provision.readthedocs.io/en/latest/doc/ui.html#prefs">UI Help</a> | <a target="_blank" href="/swagger-ui/#/prefs">API Help</a>
      </span>
      </h2>
      <table className="fullwidth input-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Value</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {Object.keys(this.state.prefs).map((val, i) =>
            <tr key={i}>
              <td>{val}</td>
              <td>
              {( val.indexOf("BootEnv") > 0 && val != "debugBootEnv"
                ?  <select
                    name={val}
                    type="bool"
                    value={this.state.prefs[val]}
                    onChange={this.handleChange}>
                    { this.props.bootenvs.map((v) =>
                      <option key={v} value={v}>{v}</option>
                    )}
                  </select>
                : <input
                    type="text"
                    name={val}
                    size="10"
                    value={this.state.prefs[val]}
                    onChange={this.handleChange} />
              )}
              </td>
              <td className="icon-buttons">
                {(this.state.updated && Object.keys(this.state.prefs).length-1 == i ? <button onClick={this.updatePrefs} className="icon-button">
                  save
                  <span className="tooltip">Save</span>
                </button> : '')}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
    );
  }
}


class BootEnv extends React.Component {

  constructor(props) {
    super(props);

    this.toggleExpand = this.toggleExpand.bind(this);
    this.handleChange = this.handleChange.bind(this);
    this.update = this.update.bind(this);
    this.remove = this.remove.bind(this);
    this.changeTemplate = this.changeTemplate.bind(this);
    this.addTemplate = this.addTemplate.bind(this);
    this.removeTemplate = this.removeTemplate.bind(this);
  }

  // expands this subnet
  toggleExpand() {
    var bootenv = this.props.bootenv;
    bootenv._expand = !bootenv._expand;
    this.props.change(this.props.index, bootenv);
  }

  // called to make the post/put request that updates the subnet
  update() {
    this.props.update(this.props.index);
  }

  // makes the delete request to remove the subnet
  remove() {
    this.props.remove(this.props.index);
  }

  // called when an input changes
  handleChange(event) {
    var val = event.target.value;
    if(event.target.type === "number" && val && typeof val !== 'undefined')
      val = parseInt(val);
    var bootenv = this.props.bootenv;
    bootenv[event.target.name] = val;
    bootenv._edited = true;

    this.props.change(this.props.index, bootenv);
  }

  changeTemplate(event, i) {
    this.props.bootenv.Templates[i][event.target.name] = event.target.value;
    this.props.bootenv._edited = true;
    this.props.change(this.props.index, this.props.bootenv);
  }
  
  addTemplate(event) {
    this.props.bootenv.Templates.push({
      Name: "",
      Path: "",
      ID: ""
    });
    this.props.bootenv._edited = true;
    this.props.change(this.props.index, this.props.bootenv);
  }

  removeTemplate(event, i) {
    this.props.bootenv.Templates.splice(i, 1);
    this.props.bootenv._edited = true;
    this.props.change(this.props.index, this.props.bootenv);
  }


  // renders the element
  render() {
    var bootenv = JSON.parse(JSON.stringify(this.props.bootenv));
    return (
      <tbody 
        className={(bootenv.updating ? 'updating-content' : '') + " " + (bootenv._expand ? 'expanded' : '') + ' ' + (bootenv._error ? 'error' : (bootenv._new ? 'new' : (bootenv._edited ? 'edited' : '')))}>
        <tr>
          <td>
            {bootenv._new ? <input
              type="text"
              name="Name"
              size="8"
              placeholder="eth0"
              value={bootenv.Name}
              onChange={this.handleChange}/> : bootenv.Name}
          </td>
          <td>
            {bootenv.Available ? "Yes" : "Error"}
          </td>
          <td>
            <a href={bootenv.OS.IsoUrl}>{bootenv.OS.IsoFile}</a>
          </td>
          <td className="icon-buttons">
            {bootenv._new || bootenv._edited ? 
            <button onClick={this.update} className="icon-button">
              save
              <span className="tooltip">{bootenv._new ? 'Add' : 'Save'}</span>
            </button> : ''}

            <button onClick={this.remove} className="icon-button">
              delete
              <span className="tooltip">Remove</span>
            </button>
            <button onClick={this.props.copy} className="icon-button">
              content_copy
              <span className="tooltip">Copy</span>
            </button>
          </td>
        </tr>
        <tr>
          <td colSpan="7">
            {bootenv._expand ? (<div>
              <table>
                <tbody>
                  <tr>
                    <td style={{textAlign: "right", fontWeight: "bold"}}>
                      Iso Name
                    </td>
                    <td>
                      <input
                        type="text"
                        size="30"
                        value={bootenv.OS.IsoFile}
                        onChange={(event) => {
                          bootenv._edited = true;
                          bootenv.OS.IsoFile = event.target.value;
                          this.props.change(this.props.index, bootenv);
                        }}/>
                    </td>
                  </tr>
                  <tr>
                    <td style={{textAlign: "right", fontWeight: "bold"}}>
                      Iso URL
                    </td>
                    <td>
                      <input
                        type="text"
                        size="30"
                        value={bootenv.OS.IsoUrl}
                        onChange={(event) => {
                          bootenv._edited = true;
                          bootenv.OS.IsoUrl = event.target.value;
                          this.props.change(this.props.index, bootenv);
                        }}/>
                    </td>
                  </tr>

                </tbody>
              </table>
              
              <h2>Templates</h2>
              <table>
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Path</th>
                    <th>ID</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {bootenv.Templates.map((val, i) =>
                    <tr key={i}>
                      <td>
                        <input
                          type="text"
                          size="15"
                          name="Name"
                          value={val.Name}
                          onChange={(e)=>this.changeTemplate(e, i)}/>
                      </td>
                      <td>
                        <input
                          type="text"
                          size="30"
                          name="Path"
                          value={val.Path}
                          onChange={(e)=>this.changeTemplate(e, i)}/>
                      </td>
                      <td>
                        <input
                          type="text"
                          size="20"
                          name="ID"
                          value={val.ID}
                          onChange={(e)=>this.changeTemplate(e, i)}/>
                      </td>
                      <td className="icon-buttons">
                        <button onClick={(e)=>$.getJSON("../api/v3/templates/" + val.ID, this.props.preview)} className="icon-button">
                          open_in_new
                          <span className="tooltip">Preview</span>
                        </button>
                        <button onClick={(e)=>this.removeTemplate(e, i)} className="icon-button">
                          delete
                          <span className="tooltip">Remove</span>
                        </button>
                      </td>
                    </tr>
                  )}
                  <tr>
                    <td colSpan="4" style={{textAlign: "center"}}>
                      <button onClick={this.addTemplate}>Add Template</button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>): <span/>}
            {bootenv._error && <div>
              <h2><span className="material-icons">error</span>{bootenv._errorMessage}</h2>
            </div>}
            <div className="expand" onClick={this.toggleExpand}>
              {bootenv._expand ? <span className="material-icons">expand_less</span> : <span className="material-icons">expand_more</span>}
            </div>
          </td>
        </tr>
      </tbody>
    );
  }
}

class BootEnvs extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      bootenvs: [],
      templatePreview: undefined,
    };

    this.componentDidMount = this.componentDidMount.bind(this);
    this.update = this.update.bind(this);
    this.addBootEnv = this.addBootEnv.bind(this);
    this.updateBootEnv = this.updateBootEnv.bind(this);
    this.removeBootEnv = this.removeBootEnv.bind(this);
    this.changeBootEnv = this.changeBootEnv.bind(this);
    this.previewTemplate = this.previewTemplate.bind(this);
  }
  
  // gets the bootenv json from the api
  getBootEnvs() {
    return new Promise((resolve, reject) => {

      // get the interfaces from the api
      $.getJSON("../api/v3/bootenvs", data => {
        resolve({
          bootenvs: data,
        });
      }).fail(err => {
        reject("Failed getting BootEnvs");
      });
    });
  }

  previewTemplate(templatePreview) {
    this.setState({templatePreview});
  }

  componentDidMount() {
    this.update();
  }

  update() {
    this.getBootEnvs().then(data => {
      this.setState({
        bootenvs: data.bootenvs,
        templatePreview: undefined,
      });
    }, err => {
    });
  }

  // called to create a new subnet    
  // allows some data other than defaults to be passed in   
  addBootEnv(template) {    
    var bootenv = {   
      _new: true,
      Name: '',
      Description: '',
      OS: {
        Name: '',
        Family: '',
        Codename: '',
        Version: '',
        IsoFile: '',
        IsoSha256: '',
        IsoUrl: ''
      },
      Templates: [],
      Kernel: "",
      Initrds: [],
      RequiredParams: [],
      Available: true,
      Errors: []
    };

    // merge the template into our bootenv if we have one
    if(typeof template !== "undefined") {
      for(var key in template) {
        if(key[0] === "_")
          continue;

        bootenv[key] = template[key];
      }
    }

    // update the state
    this.setState({
      bootenvs: this.state.bootenvs.concat(bootenv)
    });
  }

  // makes the post/put request to update the bootenv
  updateBootEnv(i) {
    var bootenv = this.state.bootenvs[i];
    bootenv.updating = true;
    this.setState({bootenv: this.state.bootenvs});

    $.ajax({
      type: (bootenv._new ? "POST" : "PUT"),
      dataType: "json",
      contentType: "application/json",
      url: "/api/v3/bootenvs" + (bootenv._new ? "" : "/" + bootenv.Name),
      data: JSON.stringify(bootenv)
    }).done((resp) => {
      
      // update the bootenvs list with our new interface
      var bootenvs = this.state.bootenvs.concat([]);

      resp.updating = false;
      resp._edited = false;
      resp._new = false;
      resp._error = false;
      resp._errorMessage = '';
      
      //  update the state
      bootenvs[i] = resp;
      this.setState({
        bootenvs: bootenvs
      });

    }).fail((err) => {
      
      var bootenvs = this.state.bootenvs.concat([]);
      var bootenv = bootenvs[i];
      bootenv.updating = false;
      bootenv._error = true;

      // If our error is from the backend
      if(err.responseText) {
        var response = JSON.parse(err.responseText);
        bootenv._errorMessage = " (" + err.status + "): " + response.Messages.join(", ");
      } else { // maybe the backend is down
        bootenv._errorMessage = err.status;
      }

      this.setState({
        bootenvs: bootenvs
      });
    });
  }

  // makes the delete request to remove the subnet or just deletes the new subnet
  removeBootEnv(i) {
    var bootenvs = this.state.bootenvs.concat([]);
    var bootenv = this.state.bootenvs[i];
    if(bootenv._new) {
      bootenvs.splice(i, 1);
      this.setState({
        bootenvs: bootenvs
      });
      return;
    }
    bootenvs[i].updating = true;
    this.setState({bootenvs: bootenvs});

    $.ajax({
      type: "DELETE",
      dataType: "json",
      contentType: "application/json",
      url: "/api/v3/bootenvs/" + bootenv.Name,
    }).done((resp) => {
            // update the bootenvs list with our new interface
      var bootenvs = this.state.bootenvs.concat([]);
      bootenvs.splice(i, 1);
      this.setState({
        bootenvs: bootenvs
      });

    }).fail((err) => {
      bootenv.updating = false;
      bootenv._error = true;
      // If our error is from the backend
      if(err.responseText) {      
        var response = JSON.parse(err.responseText);
        bootenv._errorMessage = " (" + err.status + "): " + response.Messages.join(", ");
      } else { // maybe the backend is down
        bootenv._errorMessage = err.status;
      }

      this.setState({
        bootenvs: bootenvs
      });
    });
  }

  // updates the state and changes a subnet at a specified index
  changeBootEnv(i, bootenv) {
    var bootenvs = this.state.bootenvs.concat([]);
    bootenvs[i] = bootenv;
    this.setState({
      bootenvs: bootenvs
    });
  }

  render() {
    $('#bootenvCount').text(this.state.bootenvs.length);
    return (
    <div id="bootenvs" style={{paddingTop: '51px'}}>
      <h2 style={{display: 'flex', justifyContent: 'space-between'}}>
        <span className="section-head">Boot Environments</span>
        <span>
          <a target="_blank" href="http://provision.readthedocs.io/en/latest/doc/ui.html#bootenvs">UI Help</a> | <a target="_blank" href="/swagger-ui/#/bootenvs">API Help</a>
        </span>
      </h2>
      <table className="fullwidth input-table">
        <thead>
          <tr>
            <th>Environment</th>
            <th>Available</th>
            <th>Path</th>
            <th></th>
          </tr>
        </thead>
        {this.state.bootenvs.map((val, i) =>
          <BootEnv
            bootenv={val}
            update={this.updateBootEnv}
            change={this.changeBootEnv}
            remove={this.removeBootEnv}
            preview={this.previewTemplate}
            copy={()=>this.addBootEnv(this.state.bootenvs[i])}
            key={i}
            index={i} />
        )}
        <tfoot>
          <tr>
            <td colSpan="4" style={{textAlign: "center"}}>
              <button onClick={()=>this.addBootEnv({})}>New BootEnv</button>
            </td>
          </tr>
        </tfoot>
      </table>
      {this.state.templatePreview && <div className="card floating overlay">
        <div className="card-toolbar">
          <h2>{this.state.templatePreview.ID || 'No ID Set'}</h2>
          <button onClick={()=>this.setState({templatePreview: undefined})} className="icon-button flat">
            close
          </button>
        </div>
        <div className="card-content">
          <pre>{this.state.templatePreview.Contents || 'No Content'}</pre>
        </div>
      </div>}
    </div>
    );
  }
}

class Page extends React.Component {

  constructor(props) {
    super(props);

    this.state = {
      access: false,
      bootenvs: []
    };

    this.onAccessChange = this.onAccessChange.bind(this);
    this.update = this.update.bind(this);
    //this.handleChange = this.handleChange.bind(this);
  }
  
  // get the page and interfaces once this component mounts
  componentDidMount() {
  }

  update() {
    var page = this;
    console.log("Updating");
    console.log(this);
    $.getJSON("../api/v3/bootenvs", data => {
      var bootenvs = [];
      // filter bootenvs
      for(var key in data) {
        if (data[key].Available)
          bootenvs.push(data[key].Name)
      }

      // update each ref
      page.setState({bootenvs: bootenvs}, () =>
        _.each(page.refs, ref => ref.update())
      );
    }).fail(() => {
    });
  }

  // called when an input changes
  onAccessChange(access, bootenvs) {
    this.setState({access: access, bootenvs: bootenvs});
  }

  render() {
    const access = this.state.access;
    const bootenvs = this.state.bootenvs;
    $('#navcontrols').css("display", access ? "flex" : "none");
    if (access) {
      return (
        <div id="swagger-ui-container" className="swagger-ui-wrap">
          <Subnets
            ref="subnets"
            onAccessChange={this.onAccessChange} />
          <hr/>
          <BootEnvs
            ref="bootenvs"
            onAccessChange={this.onAccessChange} />
          <hr/>
          <Prefs
            ref="prefs"
            bootenvs={bootenvs}
            onAccessChange={this.onAccessChange} />
          <hr/>
          <Machines
            ref="machines"
            bootenvs={bootenvs}
            onAccessChange={this.onAccessChange} />
        </div>
      );
    }
    return (
      <div>
        <center>
          <Token
            access={access}
            onAccessChange={this.onAccessChange} />
        </center>
      </div>
    );
  }
}

window.Provisioner = ReactDOM.render(<Page/>, document.getElementById('page'));
