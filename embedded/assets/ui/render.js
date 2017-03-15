/* jshint esversion: 6 */

class Subnet extends React.Component {

  constructor(props) {
    super(props);

    this.state = this.props.subnet;

    this.handleChange = this.handleChange.bind(this);
  }

  handleChange(event) {
    this.setState({
      [event.target.name]: event.target.value,
      _edited: true
    });
  }

  render() {
    var subnet = this.state;
    return (
      <tbody>
        <tr>
          <td>
            <input
              type="text"
              name="Name"
              size="8"
              placeholder="eth0"
              value={subnet.Name}
              onChange={this.handleChange}/>
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
            &amp;
            <input
              type="number"
              name="ReservedLeaseTime"
              style={{width: "50px"}}
              placeholder="7200"
              value={subnet.ReservedLeaseTime}
              onChange={this.handleChange}/>
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
          <td>{subnet._new ? 'Add' : (subnet._edited ? 'Update' : '')}</td>
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
  
  getSubnets() {
    return new Promise((resolve, reject) => {
      var subnets = {};
      var interfaces = {};

      $.getJSON("../api/v3/interfaces", data => {
        for(var key in data) {
          var iface = data[key];
          iface.broadcast = true;
          interfaces[iface.Name] = iface;
        }

        // add interfaces to subnets list
        $.getJSON("../api/v3/subnets", data => {
          for(var key in data) {
            var subnet = data[key];
            subnet.broadcast = typeof interfaces[subnet.Name] !== 'undefined';
            delete interfaces[subnet.Name];
            subnets[subnet.Name] = subnet;
          }

          resolve({
            interfaces: interfaces,
            subnets: subnets
          });
        });
      });
    });
  }

  componentDidMount() {
    this.getSubnets().then(data => {
      this.setState({
        subnets: Object.keys(data.subnets).map(k => data.subnets[k]),
        interfaces: Object.keys(data.interfaces).map(k => data.interfaces[k])
      });
    });
  }

  addSubnet(template) {
    var subnet = {
      _new: true,
      ActiveLeaseTime: 0,
      ReservedLeaseTime: 7200,
      OnlyReservations: true,
    };
    if(typeof template !== "undefined") {
      for(var key in template) {
        subnet[key] = template[key];
      }
    }
    this.setState({
      subnets: this.state.subnets.concat(subnet)
    });
  }

  updateSubnet(subnet) {

  }

  render() {
    return (
    <div>
      <table className="fullwidth">
        <thead>
          <tr>
            <th>Name/NIC</th>
            <th>Type</th>
            <th>Subnet</th>
            <th>Reservations</th>
            <th>Active &amp; Reserved Lease Time</th>
            <th>Range</th>
            <th></th>
          </tr>
        </thead>
        {this.state.subnets.map(
          (val, i) => <Subnet subnet={val} update={this.updateSubnet} key={i} />
        )}
        <tfoot>
          <tr>
            <td colSpan="7" style={{textAlign: "center"}}>
              <button onClick={this.addSubnet}>New Subnet</button>
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
