/* jshint esversion: 6 */

class Subnet extends React.Component {

  constructor(props) {
    super(props);

    this.state = this.props.subnet;

    this.handleChange = this.handleChange.bind(this);
    this.handleOptionChange = this.handleOptionChange.bind(this);
    this.update = this.update.bind(this);
  }

  // gets the name of an option from its code
  getCodeName(code) {
    var codes = {
      1: "Subnet Mask",
      3: "Default Gateway",
      6: "DNS Server",
      15: "Domain Name",
      28: "Broadcast",
      67: "Next Boot"
    };
    return codes[code] || "Code " + code;
  }

  // called to make the post/put request that updates the subnet
  update() {
    this.props.update(this.state, this);
  }

  // called when an input changes
  handleChange(event) {
    var val = event.target.value;
    if(event.target.type === "number" && val && typeof val !== 'undefined')
      val = parseInt(val);
    if(event.target.type === "select-one") {
      val = val === "true";
    }
    this.setState({
      [event.target.name]: val,
      _edited: true
    });
  }

  // called when an option input is changed
  handleOptionChange(event) {
    var options = this.state.Options;
    options[event.target.name].Value = event.target.Value;

    this.setState({
      Options: options,
      _edited: true
    });
  }

  // renders the element
  render() {
    var subnet = this.state;
    return (
      <tbody 
        className={(this.state.updating) ? 'updating-content' : ''}
        style={{
          position: "relative",
          backgroundColor: (this.state.error ? '#fdd' : (this.state._new ? "#dfd" : (this.state._edited ? "#eee" : "#fff"))),
          borderBottom: "thin solid #ddd"
        }}>
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
          <td>{subnet.broadcast ? "broadcast" : "relayed"}</td>
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
            <input
              type="text"
              name="ActiveStart"
              size="12"
              placeholder="10.0.0.0"
              value={subnet.ActiveStart}
              onChange={this.handleChange}/>
            ...
            <input
              type="text"
              name="ActiveEnd"
              size="12"
              placeholder="10.0.0.255"
              value={subnet.ActiveEnd}
              onChange={this.handleChange}/>
          </td>
          <td>{subnet._new ? <button onClick={this.update}>Add</button> :
              (subnet._edited ? <button onClick={this.update}>Update</button> : '')}</td>
        </tr>
        <tr>
          <td colSpan="7">
            {subnet._expand ? (<div>
              <h2>Options</h2>
              <table>
                <tbody>
                  {subnet.Options.map((val, i) =>
                    <tr key={i}>
                      <td style={{textAlign: "right", fontWeight: "bold"}}>{this.getCodeName(val.Code)}</td>
                      <td>
                        <input
                          type="text"
                          name={i}
                          value={val.Value}
                          onChange={this.handleOptionChange}/>
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>): <span/>}
            {this.state.error && <div>
              <h2>{this.state.errorMessage}</h2>
            </div>}
            <div className="expand" onClick={()=>this.setState({_expand: !subnet._expand})}>
              {subnet._expand ? <span>&#x25B4;</span> : <span>&#x25BE;</span>}
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
    this.addSubnet = this.addSubnet.bind(this);
    this.updateSubnet = this.updateSubnet.bind(this);
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
          iface.broadcast = true;
          interfaces[iface.Name] = iface;
        }

        // add get subnets from the api and update the state
        $.getJSON("../api/v3/subnets", data => {
          for(var key in data) {
            var subnet = data[key];
            subnet.broadcast = typeof interfaces[subnet.Name] !== 'undefined';
            subnets[subnet.Name] = subnet;
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
      });;
    });
  }

  // get the subnets and interfaces once this component mounts
  componentDidMount() {
    this.getSubnets().then(data => {
      this.setState({
        subnets: Object.keys(data.subnets).map(k => data.subnets[k]),
        interfaces: Object.keys(data.interfaces).map(k => data.interfaces[k])
      });
    });
  }

  // called to create a new subnet
  // allows some data other than defaults to be passed in
  addSubnet(template) {
    var subnet = {
      _new: true,
      ActiveLeaseTime: 60,
      ReservedLeaseTime: 7200,
      OnlyReservations: true,
      Strategy: "MAC",
      Options: [
        {Code: 3, Value: ''},
        {Code: 6, Value: ''},
        {Code: 15, Value: ''},
        {Code: 67, Value: ''}
      ]
    };

    // merge the template into our subnet if we have one
    if(typeof template !== "undefined") {
      for(var key in template) {
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
  updateSubnet(subnet, elem) {
    elem.setState({updating: true});
    console.log("posting: ",subnet)

    $.ajax({
      type: (subnet._new ? "POST" : "PUT"),
      dataType: "json",
      contentType: "application/json",
      url: "/api/v3/subnets" + (subnet._new ? "" : "/" + subnet.Name),
      data: JSON.stringify(subnet)
    }).done((resp) => {
      console.log('success,', elem.props.index, resp);
      
      resp.broadcast = typeof this.state.interfaces[resp.Name] !== 'undefined';
      // update the subnets list with our new interface
      var subnets = this.state.subnets.concat([]);
      subnets[elem.props.index] = resp;
      this.setState({
        subnets: subnets
      });

      // stop editing this subnet and update the state
      resp.updating = false;
      resp._edited = false;
      resp._new = false;
      resp.error = false;
      resp.errorMessage = '';
      elem.setState(resp);

    }).fail((err) => {
      console.error('fail', err);

      // If our error is from the backend
      if(err.responseText) {      
        var response = JSON.parse(err.responseText);
        elem.setState({updating: false, error: true, errorMessage: "Error (" + err.status + "): " + response.Messages.join(", ")});

      } else { // maybe the backend is down
        elem.setState({updating: false, error: true, errorMessage: err.status});
      }
    });
  }

  render() {
    return (
    <div>
      <table className="fullwidth input-table">
        <thead>
          <tr>
            <th>Name/NIC</th>
            <th>Type</th>
            <th>Subnet</th>
            <th>Reservations</th>
            <th>Active &amp; Reserved Lease</th>
            <th>Range</th>
            <th></th>
          </tr>
        </thead>
        {this.state.subnets.map(
          (val, i) => <Subnet subnet={val} update={this.updateSubnet} key={i} index={i} />
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
          {this.state.interfaces.map(val => 
            val.Addresses.map(subval =>
              <a key={val.Name} className="interface-pair" onClick={()=>this.addSubnet({Name: val.Name, Subnet: subval})}>
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

ReactDOM.render(<Subnets />, subnets)
